//
// Copyright (c) SAS Institute Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package signjar

import (
	"bytes"
	"crypto"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sassoftware/relic/config"
	"github.com/sassoftware/relic/lib/x509tools"
)

// See https://docs.oracle.com/javase/8/docs/technotes/guides/jar/jar.html#JAR_Manifest

const (
	metaInf      = "META-INF/"
	manifestName = metaInf + "MANIFEST.MF"
)

type FilesMap struct {
	Main  http.Header
	Order []string
	Files map[string]http.Header
}

func ParseManifest(manifest []byte) (files *FilesMap, err error) {
	sections, err := splitManifest(manifest)
	if err != nil {
		return nil, err
	}
	files = &FilesMap{
		Order: make([]string, 0, len(sections)-1),
		Files: make(map[string]http.Header, len(sections)-1),
	}
	for i, section := range sections {
		if i > 0 && len(section) == 0 {
			continue
		}
		hdr, err := parseSection(section)
		if err != nil {
			return nil, err
		}
		if i == 0 {
			files.Main = hdr
		} else {
			name := hdr.Get("Name")
			if name == "" {
				return nil, errors.New("manifest has section with no \"Name\" attribute")
			}
			files.Order = append(files.Order, name)
			files.Files[name] = hdr
		}
	}
	return files, nil
}

func (m *FilesMap) Dump() []byte {
	var out bytes.Buffer
	writeSection(&out, m.Main, "Manifest-Version")
	for _, name := range m.Order {
		section := m.Files[name]
		if section != nil {
			writeSection(&out, section, "Name")
		}
	}
	return out.Bytes()
}

func splitManifest(manifest []byte) ([][]byte, error) {
	sections := make([][]byte, 0)
	for len(manifest) != 0 {
		i1 := bytes.Index(manifest, []byte("\r\n\r\n"))
		i2 := bytes.Index(manifest, []byte("\n\n"))
		var idx int
		if i1 < 0 {
			if i2 < 0 {
				return nil, errors.New("trailing bytes after last newline")
			}
			idx = i2 + 2
		} else {
			idx = i1 + 4
		}
		section := manifest[:idx]
		manifest = manifest[idx:]
		sections = append(sections, section)
	}
	return sections, nil
}

func parseSection(section []byte) (http.Header, error) {
	section = bytes.Replace(section, []byte("\r\n"), []byte{'\n'}, -1)
	section = bytes.Replace(section, []byte("\n "), []byte{}, -1)
	keys := bytes.Split(section, []byte{'\n'})
	hdr := make(http.Header)
	for _, line := range keys {
		if len(line) == 0 {
			continue
		}
		idx := bytes.IndexRune(line, ':')
		if idx < 0 {
			return nil, errors.New("jar manifest is malformed")
		}
		key := strings.TrimSpace(string(line[:idx]))
		value := strings.TrimSpace(string(line[idx+1:]))
		hdr.Set(key, value)
	}
	return hdr, nil
}

func hashSection(hash crypto.Hash, section []byte) string {
	d := hash.New()
	d.Write(section)
	return base64.StdEncoding.EncodeToString(d.Sum(nil))
}

// Transform a MANIFEST.MF into a *.SF by digesting each section with the
// specified hash
func DigestManifest(manifest []byte, hash crypto.Hash, sectionsOnly, apkV2 bool) ([]byte, error) {
	sections, err := splitManifest(manifest)
	if err != nil {
		return nil, err
	}
	hashName := x509tools.HashNames[hash]
	if hashName == "" {
		return nil, errors.New("unsupported hash type")
	}
	var output bytes.Buffer
	writeAttribute(&output, "Signature-Version", "1.0")
	writeAttribute(&output, hashName+"-Digest-Manifest-Main-Attributes", hashSection(hash, sections[0]))
	if !sectionsOnly {
		writeAttribute(&output, hashName+"-Digest-Manifest", hashSection(hash, manifest))
	}
	writeAttribute(&output, "Created-By", fmt.Sprintf("%s (%s)", config.UserAgent, config.Author))
	if apkV2 {
		writeAttribute(&output, "X-Android-APK-Signed", "2")
	}
	output.WriteString("\r\n")
	for _, section := range sections[1:] {
		hdr, err := parseSection(section)
		if err != nil {
			return nil, err
		}
		name := hdr.Get("Name")
		if name == "" {
			return nil, errors.New("File section was missing Name attribute")
		}
		writeAttribute(&output, "Name", name)
		writeAttribute(&output, hashName+"-Digest", hashSection(hash, section))
		output.WriteString("\r\n")
	}
	return output.Bytes(), nil
}

const maxLineLength = 70

// Write a key-value pair, wrapping long lines as necessary
func writeAttribute(out io.Writer, key, value string) {
	line := []byte(fmt.Sprintf("%s: %s", key, value))
	for i := 0; i < len(line); i += maxLineLength {
		j := i + maxLineLength
		if j > len(line) {
			j = len(line)
		}
		if i != 0 {
			out.Write([]byte{' '})
		}
		out.Write(line[i:j])
		out.Write([]byte("\r\n"))
	}
}

func writeSection(out io.Writer, hdr http.Header, first string) {
	value := hdr.Get(first)
	if value != "" {
		writeAttribute(out, first, value)
	}
	for key, values := range hdr {
		if key == first {
			continue
		}
		for _, value := range values {
			writeAttribute(out, key, value)
		}
	}
	out.Write([]byte("\r\n"))
}
