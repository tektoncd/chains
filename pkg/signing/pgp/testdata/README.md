# Test Data

This directory contains test data used to test our openpgp usage.
The private/public keypair are valid, but should only be used for tests.

They were generated with:

```
$ gpg --gen-key
Real name: Tekton Unit Tests
Email address: testing@tekton.dev
You selected this USER-ID:
    "Tekton Unit Tests <testing@tekton.dev>"
$ gpg --export "Tekton Unit Tests" > pgp.private-key
$ gpg --export-secret-key "Tekton Unit Tests" --armor > pgp.public-key
$ echo -n "testing123" > pgp.passphrase
```

The passphrase is: `testing123`.
