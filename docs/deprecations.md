<!--
---
linkTitle: "Deprecations"
weight: 5000
---
-->

# Deprecations

- [Introduction](#introduction)
- [Deprecation Table](#deprecation-table)

## Introduction

This doc provides a list of features in Tekton Chains that are
being deprecated.

Deprecations will follow this timeline:
- Deprecation announcement is made during a release
- Feature is removed two releases later

So, if a feature is deprecated at v0.1.0, then it would be removed in v0.3.0.

## Deprecation Table

| Feature Being Deprecated  | Deprecation Announcement  | API Compatibility Policy | Earliest Date or Release of Removal |
| ------------------------- | ------------------------- | ------------------------ | ----------------------------------- |
| Support for PipelineResources was removed, see [TEP0074](https://github.com/tektoncd/community/blob/main/teps/0074-deprecate-pipelineresources.md) | [v0.16.0 (https://github.com/tektoncd/chains/releases/tag/v0.16.0) | Alpha | v0.16.0 |
| [`tekton-provenance` format is deprecated](https://github.com/tektoncd/chains/issues/293)  | [v0.6.0](https://github.com/tektoncd/pipeline/releases/tag/v0.6.0) | Alpha | v0.8.0 |
