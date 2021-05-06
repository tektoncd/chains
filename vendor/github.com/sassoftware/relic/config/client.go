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

package config

import "crypto/x509"

func (cl *ClientConfig) Match(incoming []*x509.Certificate) (bool, error) {
	if cl.certs == nil || len(incoming) == 0 {
		return false, nil
	}
	leaf := incoming[0]
	intermediates := incoming[1:]
	ipool := x509.NewCertPool()
	for _, cert := range intermediates {
		ipool.AddCert(cert)
	}
	_, err := leaf.Verify(x509.VerifyOptions{
		Roots:         cl.certs,
		Intermediates: ipool,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	})
	if err == nil {
		return true, nil
	} else if _, ok := err.(x509.UnknownAuthorityError); ok {
		return false, nil
	}
	return false, err
}
