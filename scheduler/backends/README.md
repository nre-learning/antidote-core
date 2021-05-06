# Antidote Backends

As of the initial release, Antidote had a somewhat tight integration with Kubernetes to provide the back-end compute infrastructure for running lesson resources/endpoints. While the Antidote API itself has been quite abstract from the beginning (API clients don't need to know anything about Kubernetes), the Antidote scheduler still translated from these abstract primitives statically to Kubernetes API calls to make lessons work.

In 2021, this portion of the Antidote service was extended to allow for the possibility to integrate with other infrastructure solutions, such as OpenStack, or cloud providers. To facilitate this, the codebase was restructured so that the various back-end providers could be defined in the `scheduler/backends/` directory. In addition, a high-level Go interface was defined to provide some structure for the minimum set of functions that a backend needs to provide so that the Antidote scheduler can use it to provision resources.

This document aims to describe the complete set of requirements and conventions that must be followed by all backend providers, in order to provide a consistent experience to users of an Antidote-based system regardless of what infrastructure happens to be used behind the scenes. All contributors are encouraged to read this document fully, as any contributions must follow it.

> As the `kubernetes` backend was developed first and iterated on for years, contributors are heavily encouraged to refer to that backend for guidance on best practices. There are, of course, kubernetes-specific implementation details there, but there are also plenty of lessons learned baked between the lines that shouldn't be disregarded.

> What follows below was written as a first-attempt at capturing backend requirements. Before, there was no need for this, given the single-backend nature of Antidote up to now. As a result, while great care was taken to make this as exhaustive as possible, it's almost certain that some things were left off. Contributors should still expect for a lengthy and dynamic review process, as adding a backend to Antidote is likely to require several iteration cycles to get right.

## Expected Behaviors

Before getting into the implementation details, there are some behaviors that are considered "core functionality" for Antidote, and it is the responsibility of all backends to ensure they're fully supported. These items should not depend on which backend happens to be running behind the scenes.

1. DNS lookups by endpoint name
2. `/antidote` (or otherwise configured) directory mapping
3. All presentation types, using the conventions established (e.g. the subdomains for HTTP endpoints will need L7 routing)

## Implementation Type and Directory Structure

All backends are compiled into the same binary
Need to add a directory and a go file with the same name
need to add an option to the configuration
Need to augment the conditional that chooses the backend based on config

Each backend should have a high-level struct named according to the infrastructure it integrates with, followed by `Backend`. For instance, the Kubernetes backend is called `KubernetesBackend`. This struct can have whatever fields are needed for the backend to maintain its own state, but must include the three fields shown in the below example:

```go
type KubernetesBackend struct {
	Config    config.AntidoteConfig
	Db        db.DataManager
	BuildInfo map[string]string

    // other fields specific to this backend can follow here.
}
```

Backend-specific struct fields can be used to maintain some state specific to that backend, (like a handle on an API client, etc), however the `DataManager` passed to the backend by the schedule should be used for **all** Antidote-specific state like livelessons, livesessions, etc. It is highly encouraged that contributors review the package in `db/` first, as this is where state management functions are defined, and their use is required.

There must also be a factory function in the same file that returns an initialized instance of this type, ready to be used by the scheduler. All setup activities should take place here, and this function should receive types that will satisfy the field requirements of the struct defined above. See `NewKubernetesBackend` for an example of this.


## The "AntidoteBackend" Interface

The Antidote scheduler interacts with a backend using a small handful of high-level functions. These functions are defined in a Go interface called [`AntidoteBackend`](https://github.com/nre-learning/antidote-core/blob/master/scheduler/scheduler.go). 

This interface is intentionally kept light, as there are only a handful of high-level functions that all backends must implement in order for Antidote to work with them. However, satisfying this interface is non-optional. The struct described in the previous section must fully implement `AntidoteBackend`.

In addition to satisfying this interface, there are several implementation details that must be adhered to. These details are documented as a docstring above each function reference in the `AntidoteBackend` interface. so please read those comments. There are also implementation details that apply to ALL functions in this interface, which are captured as a docstring above the interface. You should read all these comments before working on a backend.

## Additional Implementation Details

Functions should be instrumented properly as done in the kubernetes backend.




## Misc

- `antidote-images`, while opinionated towards Kubernetes in their current form, are a necessary component to antidote. There is a need to have platform-versioned images for any back-end infrastructure that `antidoted` can just use to run configuration scripts, jupyter notebooks, etc. It is highly recommended that any backend development effort augment that repository to add build scripts for each of those images relevant to that backend's infrastructure target, and then using those images in the same way that the kubernetes backend does currently.
- It's likely that additional backends will require this anyways, but support for private image sources is not optional. There should at least be an option to supply credentials when accessing an endpoint image to be used by a backend.