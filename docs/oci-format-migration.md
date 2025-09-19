# OCI Storage Format Migration Guide

This guide provides comprehensive instructions for migrating from the deprecated `storage.oci.referrers-api` boolean configuration to the new flexible `storage.oci.format` enum-based configuration in Tekton Chains.

## Overview

Tekton Chains has refactored its OCI storage backend to support three distinct storage formats, replacing the binary choice of the `storage.oci.referrers-api` boolean setting with a more flexible and precise configuration system.

### What Changed

**Before (Deprecated):**
```yaml
storage.oci.referrers-api: true  # or false
```

**After (New):**
```yaml
storage.oci.format: "legacy"           # or "referrers-api" or "protobuf-bundle"
```

## Migration Path

### Automatic Migration

Tekton Chains automatically migrates existing configurations during startup:

| Old Configuration | New Configuration | Behavior |
|-------------------|-------------------|----------|
| `storage.oci.referrers-api: false` | `storage.oci.format: "legacy"` | Tag-based storage (default) |
| `storage.oci.referrers-api: true` | `storage.oci.format: "protobuf-bundle"` | Protobuf bundle format |
| Not set | `storage.oci.format: "legacy"` | Tag-based storage (default) |

### Migration Warnings

When using deprecated configuration, you'll see warnings in the Chains controller logs:
```
WARN storage.oci.referrers-api is deprecated, use storage.oci.format instead
INFO Migrated storage.oci.referrers-api=false to storage.oci.format=legacy
```

To remove these warnings, update your configuration manually.

## Format Comparison

### Legacy Format (`legacy`)

**Characteristics:**
- **Storage Mechanism**: Tag-based storage (`<image>:sha256-<digest>.sig`, `<image>:sha256-<digest>.att`)
- **Serialization Format**: DSSE (Dead Simple Signing Envelope)
- **Registry Compatibility**: All OCI-compliant registries
- **Tooling Compatibility**: Full backward compatibility with existing tools
- **Registry Impact**: Creates additional tags for each signature and attestation

**When to Use:**
- Production deployments requiring maximum compatibility
- Existing workflows that depend on tag-based signatures
- Registries that don't support OCI 1.1 features
- Policy engines expecting DSSE format

**Example:**
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: chains-config
  namespace: tekton-chains
data:
  storage.oci.format: "legacy"
```

**Signature Artifacts Created:**
- Signature: `registry.io/image:sha256-abc123.sig`
- Attestation: `registry.io/image:sha256-abc123.att`

### Referrers API Format (`referrers-api`)

**Characteristics:**
- **Storage Mechanism**: OCI 1.1 Referrers API
- **Serialization Format**: DSSE (Dead Simple Signing Envelope) - same as legacy
- **Registry Compatibility**: OCI 1.1 compatible registries
- **Tooling Compatibility**: Compatible with DSSE-aware tools and policy engines
- **Registry Impact**: Uses referrers relationships, significantly reduces tag proliferation

**When to Use:**
- OCI 1.1 compatible registries
- Environments where tag proliferation is a concern
- Existing DSSE-based policy engines and verification tools
- Migration path from legacy while maintaining format compatibility

**Example:**
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: chains-config
  namespace: tekton-chains
data:
  storage.oci.format: "referrers-api"
```

**Signature Artifacts Created:**
- Signatures and attestations linked via OCI 1.1 referrers API
- No additional tags created in registry
- DSSE format maintained for tool compatibility

### Protobuf Bundle Format (`protobuf-bundle`)

**Characteristics:**
- **Storage Mechanism**: OCI 1.1 Referrers API
- **Serialization Format**: Protobuf bundle format (experimental)
- **Registry Compatibility**: OCI 1.1 compatible registries
- **Tooling Compatibility**: Requires cosign experimental features
- **Registry Impact**: Uses referrers relationships with experimental serialization

**When to Use:**
- Testing new cosign experimental features
- Development and testing environments
- Future-proofing for upcoming cosign capabilities
- **Not recommended for production use**

**Example:**
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: chains-config
  namespace: tekton-chains
data:
  storage.oci.format: "protobuf-bundle"
```

**Signature Artifacts Created:**
- Protobuf-serialized bundles via referrers API
- Experimental format subject to change
- Requires `COSIGN_EXPERIMENTAL=1` environment variable

## Recommended Migration Steps

### Phase 1: Update Configuration (Immediate)

1. **Remove Deprecation Warnings**

   Update your `chains-config` ConfigMap to use the new format:

   ```bash
   kubectl patch configmap chains-config -n tekton-chains --type merge -p '{
     "data": {
       "storage.oci.format": "legacy"
     }
   }'
   ```

2. **Remove Deprecated Setting**

   If you have the old setting, remove it:

   ```bash
   kubectl patch configmap chains-config -n tekton-chains --type json -p '[
     {"op": "remove", "path": "/data/storage.oci.referrers-api"}
   ]'
   ```

### Phase 2: Test Referrers API (Optional)

1. **Prerequisites Check**

   Verify your registry supports OCI 1.1:
   ```bash
   # Test registry OCI 1.1 support
   curl -H "Accept: application/vnd.oci.image.index.v1+json" \
        https://your-registry.io/v2/your-repo/referrers/sha256:digest
   ```

2. **Update to Referrers API Format**

   ```bash
   kubectl patch configmap chains-config -n tekton-chains --type merge -p '{
     "data": {
       "storage.oci.format": "referrers-api"
     }
   }'
   ```

3. **Verify Functionality**

   Run test TaskRuns/PipelineRuns and verify signatures are created correctly:
   ```bash
   # Check controller logs
   kubectl logs -n tekton-chains deployment/tekton-chains-controller

   # Look for: "Using OCI 1.1 referrers API with DSSE format"
   ```

### Phase 3: Experimental Features (Advanced)

**Warning**: Only for testing and development environments.

```bash
kubectl patch configmap chains-config -n tekton-chains --type merge -p '{
  "data": {
    "storage.oci.format": "protobuf-bundle"
  }
}'
```

## Registry Compatibility

### OCI 1.1 Referrers API Support

| Registry | OCI 1.1 Support | Recommended Format |
|----------|-----------------|-------------------|
| Docker Hub | ❌ | `legacy` |
| Google Artifact Registry | ✅ | `referrers-api` or `legacy` |
| Amazon ECR | ✅ | `referrers-api` or `legacy` |
| Azure Container Registry | ✅ | `referrers-api` or `legacy` |
| Harbor v2.5+ | ✅ | `referrers-api` or `legacy` |
| Quay.io | ✅ | `referrers-api` or `legacy` |
| JFrog Artifactory | ✅ | `referrers-api` or `legacy` |
| Distribution v2.8+ | ✅ | `referrers-api` or `legacy` |

### Registry Feature Testing

Test OCI 1.1 support in your registry:

```bash
# Test script to verify referrers API support
#!/bin/bash
REGISTRY="your-registry.io"
REPO="test-repo"
DIGEST="sha256:your-image-digest"

# Test referrers endpoint
curl -f -H "Accept: application/vnd.oci.image.index.v1+json" \
     "https://${REGISTRY}/v2/${REPO}/referrers/${DIGEST}" \
  && echo "✅ OCI 1.1 referrers API supported" \
  || echo "❌ OCI 1.1 referrers API not supported, use legacy format"
```

## Troubleshooting

### Common Issues

#### 1. Registry Doesn't Support Referrers API

**Error:**
```
ERROR writing attestations with referrers API: 404 endpoint not found
```

**Solution:**
Fall back to legacy format:
```yaml
storage.oci.format: "legacy"
```

#### 2. COSIGN_EXPERIMENTAL Not Set

**Error:**
```
ERROR experimental features not enabled
```

**Solution:**
Ensure the Chains deployment has the environment variable set:
```yaml
env:
  - name: COSIGN_EXPERIMENTAL
    value: "1"
```

#### 3. Migration Warnings Persist

**Issue:**
Deprecation warnings continue after updating configuration.

**Solution:**
1. Ensure old configuration is completely removed
2. Restart Chains controller pod
3. Check ConfigMap has been updated properly

#### 4. Policy Engine Compatibility

**Issue:**
Existing policy engines can't read new format.

**Solution:**
- Use `referrers-api` format to maintain DSSE compatibility
- Update policy engines to support OCI 1.1 referrers
- Use `legacy` format for immediate compatibility

### Verification Commands

#### Check Current Configuration
```bash
kubectl get configmap chains-config -n tekton-chains -o yaml
```

#### Monitor Controller Logs
```bash
kubectl logs -f -n tekton-chains deployment/tekton-chains-controller
```

#### Verify Signature Creation
```bash
# For legacy format - check for .sig/.att tags
cosign verify --key /path/to/key your-registry.io/image:tag

# For referrers API - check referrers endpoint
curl -H "Accept: application/vnd.oci.image.index.v1+json" \
     https://your-registry.io/v2/repo/referrers/sha256:digest
```

## Best Practices

### Production Deployments

1. **Start with Legacy Format**
   - Ensures maximum compatibility
   - Proven stability in production
   - Easy rollback path

2. **Test Before Migration**
   - Validate registry compatibility
   - Test with sample workloads
   - Verify policy engine compatibility

3. **Gradual Migration**
   - Update non-critical namespaces first
   - Monitor for issues
   - Have rollback plan ready

### Development Environments

1. **Use Referrers API Format**
   - Reduces registry clutter
   - Tests modern OCI features
   - Maintains tool compatibility

2. **Experiment with Protobuf Bundle**
   - Only in development
   - Test new cosign features
   - Provide feedback to upstream

## Advanced Configuration

### Multi-Registry Environments

For environments with mixed registry capabilities:

```yaml
# Use legacy for maximum compatibility
storage.oci.format: "legacy"
```

Future versions may support per-repository format configuration.

### Performance Considerations

| Format | Registry Performance | Verification Performance | Storage Efficiency |
|--------|---------------------|-------------------------|-------------------|
| `legacy` | Tag queries required | Standard DSSE verification | More registry storage |
| `referrers-api` | Referrers API calls | Standard DSSE verification | Efficient storage |
| `protobuf-bundle` | Referrers API calls | Experimental verification | Efficient storage |

## Future Considerations

### Upcoming Features

1. **Per-Repository Format Configuration**
   - Different formats for different repositories
   - Registry capability auto-detection

2. **Format Auto-Selection**
   - Automatic format selection based on registry capabilities
   - Fallback mechanisms

3. **Additional Formats**
   - Support for new signature formats
   - Integration with emerging standards

### Migration Timeline

- **Current**: All three formats supported
- **Next Release**: Enhanced format features
- **Future Release**: Removal of deprecated `storage.oci.referrers-api` configuration

---

For additional support, refer to the [Tekton Chains Configuration Documentation](config.md) or file issues in the [Tekton Chains repository](https://github.com/tektoncd/chains).