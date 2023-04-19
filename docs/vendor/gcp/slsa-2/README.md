# SLSA L2 Demo with Tekton
This repo shows how Tekton can be used to acheive SLSA L2 requirements.

### To run the demo
* Clone the repo.
* Edit `env.sh` to set appropriate values for running the demo.
* Run `setup.sh` which sets up the environment.
* Run `run_pipeline.sh`.
* After the pipeline run is done run `provenance_extractor.sh`.

### How the demo works
#### `setup.sh`

`setup.sh` based on the settings specified in `env.sh`:

* Creates a Kubernetes cluster on Google Cloud.
* Sets up Artifact registry on Google Cloud.
* Sets up the KMS on Google Cloud.
* Sets up the necessary Service Accounts.
* Installs Tekton Pipelines and Tekton Chains on the Kubernetes cluster.

#### `run_pipeline.sh`
`run_pipeline.sh` replaces the ENV variables in `pipeline_run.yaml` with the
values defined in `env.sh` and then runs `pipeline_run.yaml`.

#### `provenance_extractor.sh`
`provenance_extractor.sh` extracts the provenance and stores it into a file
named `provenance.json`.
