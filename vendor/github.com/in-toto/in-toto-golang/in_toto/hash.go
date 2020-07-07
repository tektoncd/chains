package in_toto

import (
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
)

/*
Hash is an interface which contains a generic compute method.
This method is implemented by several functions to return the
hash for a given data.

To add hash functions which are currently not supported one
has to create a struct, write it's definition, add it's
value to the map in createMap function and add the hash
function to the list of hash functions(if required)
*/

type Hash interface {
	Compute(contents []uint8) string
}

/*
Declaration of struct, one for each of the hash function.
*/
type sha256Hash struct{}
type sha512Hash struct{}
type md5Hash struct{}

/*
Definition of compute function for each of the hash struct
declared above.
*/

func (hash *sha256Hash) Compute(contents []uint8) string {

	hashed := sha256.Sum256(contents)
	n := fmt.Sprintf("%x", hashed)
	return n
}

func (hash *sha512Hash) Compute(contents []uint8) string {
	hashed := sha512.Sum512(contents)
	n := fmt.Sprintf("%x", hashed)
	return n
}

func (hash *md5Hash) Compute(contents []uint8) string {
	hashed := md5.Sum(contents)
	n := fmt.Sprintf("%x", hashed)
	return n
}

/*
This function returns the map containing hash function name as key and
their respective reference object as value.
*/

func createMap() map[string]interface{ Compute(contents []uint8) string } {
	mapper := map[string]interface{ Compute(contents []uint8) string }{
		"sha256": &sha256Hash{},
		"sha512": &sha512Hash{},
		"md5":    &md5Hash{},
	}
	return mapper
}

/*
This function return a list a hash functions that we want program to use
for each of the files.
*/

func createList() []string {

	hashFunc := []string{"sha256", "sha512", "md5"}
	return hashFunc
}
