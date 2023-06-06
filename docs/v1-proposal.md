# v1 client changes proposal - storage refactoring

With v1 approaching, I want to take a moment to look at changes we want to make
to the existing client libraries to better set us up for long term maintenance.

We already know that we want to reduce the external library surface of chains.
But to do this, we need to define better interfaces between components that we
expect external clients to use.

Today, I think a lot of the codebase's complexity has come from a few places:

1. Storage libraries

   Each storage type needs different pieces of data - i.e. Grafeas and OCI need
   to distinguish image signatures and attestation formats, and some clients
   need the original object to extract out information like GVKs, names,
   namespaces, etc. This has led to a organic growth of the chains libraries to
   pass the different types of data around, and a lot of typecasting and other
   generic object tricks.

   Looking at the storage interfaces, I think this data roughly boils down to:

   - The original Tekton object
   - The formatted data object
   - The signed payload + signature (with optional cert information)

   Unlike when chains first started, we now have another useful tool available
   to us: generics. I think we can use this to create clearer interfaces.

2. Dependence on the config package

   tkn depends on the chains server config, but it probably shouldn't. We should
   aim to have better ways to initialize clients for others to use.

Good news, I don't think we're far off, but we should make some changes

## Signables

At it's core, Chains is basically an ETL pipeline. We Extract artifacts from run
objects, Transform and sign them, then Load them into storage.

```go
type Signable[T any] interface {
	Extract(ctx context.Context, obj objects.TektonObject) []T{}
}
```

## Payloaders

I think payloaders are mostly in a good place, though we can introduce generics
to start creating stricter type relationships between Signables and Payloaders.

```go
type Payloader[Input any, Output BinaryMarshaler] interface {
	CreatePayload(ctx context.Context, in Input) (Output, error)
}
```

tl;dr: Some type comes in, some type comes out.

[BinaryMarshaler comes from the encoding package](https://pkg.go.dev/encoding#BinaryMarshaler),
but basically all we're aiming for here is to make sure we can get a []byte for
signing. For existing payload types, this may mean we need to wrap external
types for this functionality.

## Signers

Signers are mostly in a good spot, though we should probably just embrace []byte
instead of typecasting between string for cert details.

```go
type Signer interface {
	signature.SignerVerifier
	Cert() []byte
	Chain() []byte
}
```

## Storers

Now that we have all the other pieces defined, we can now have stricter typing
for storing:

```go
type Storer[Input any, Output any] interface {
	Store(ctx context.Context, req *StoreRequest) (*StoreResponse, error)
}

type StoreRequest[Input any, Output any] struct {
    Object objects.TektonObject
    Artifact Input
    Payload Output
    Bundle *signing.Bundle
}

type StoreResponse struct {
    // Some identifier for what we uploaded to reference later?
    ID string
}

type Bundle struct {
	Content   []byte
	Signature []byte
	Cert      []byte
	Chain     []byte
}
```

While the StoreRequest struct may not be necessary, it has a nice RPC-like
quality in that it will make it easier to add/remove fields in the future.

## Attestors

To put it all together, we can add a new type: Attestor. This is effectively
just a wrapper type around all of the other interfaces that binds the generic
types together. Because things are strictly typed, we'll know at compile
what clients are compatible with each other.

**TBD if we expose this at all** - it may remain an internal implementation
detail of chains. This is what we will generate from the Chains server config.

```go
type Attestor[Input, Output] struct {
    payloader Payloader[Input, Output]
    signer Signer
    storer Storer[Input, Output]
}
```

What this looks like in practice:

OCI Simple Signing:

```go
attestor := &Attestor[name.Digest, simple.SimpleContainerImage]{
    payloader: NewSimpleSigningPayloader(),
    signer: x509Signer,
    storer: NewSimpleOCIStorage(),
}
```

SLSA:

```go
attestor := &Attestor[TektonObject, *intoto.Statement]{
    payloader: NewSLSAPayloader(),
    signer: fulcio,
    storer: NewGCSStorage(),
}
```

Grafeas:

```go
attestor := &Attestor[TektonObject, *Occurrence]{
    payloader: NewGrafeasPayloader(),
    signer: kmsSigner,
    storer: NewGrafeasClient(),
}
```

## Final thoughts

If all goes well, this should have 0 impact on typical consumer usage of
chains - these should all be internal refactors with no change in behavior. If
our e2e start failing, we've done something wrong.
