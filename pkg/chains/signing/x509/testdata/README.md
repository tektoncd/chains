# x509 Test Data

These keys are used ONLY FOR TESTING.

They were generated with:

```shell
$ openssl ecparam -genkey -name prime256v1 > ec_private.pem
$ openssl pkcs8 -topk8 -in ec_private.pem  -nocrypt
```