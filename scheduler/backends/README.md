# Antidote Backends

As of the initial release, Antidote had a somewhat tight integration with Kubernetes to provide the back-end compute infrastructure for running lesson resources/endpoints. While the Antidote API itself has been quite abstract from the beginning (API clients don't need to know anything about Kubernetes), the Antidote scheduler still translated from these abstract primitives statically to Kubernetes API calls to make lessons work.

In 2021, this portion of the Antidote service was extended to allow for the possibility to integrate with other infrastructure solutions, such as OpenStack, or cloud providers. To facilitate this, the codebase was restructured so that the various back-end providers could be defined in the `scheduler/backends/` directory. In addition, a high-level Go interface was defined to provide some structure for the minimum set of functions that a backend needs to provide so that the Antidote scheduler can use it to provision resources.

This document aims to describe the complete set of requirements and conventions that must be followed by all backend providers, in order to provide a consistent experience to users of an Antidote-based system regardless of what infrastructure happens to be used behind the scenes. All contributors are encouraged to read this document fully, as any contributions must follow it.

> As the `kubernetes` backend was developed first and iterated on for years, contributors are heavily encouraged to refer to that backend for guidance on best practices. There are, of course, kubernetes-specific implementation details there, but there are also plenty of lessons learned baked between the lines that shouldn't be disregarded.

> This is new functionality. While this document aims to document the process of developing a new backend as exhaustively as possible, it's inevitable that there will be some gaps. Even still, the process of developing a new backend is complex. Contributors should expect quite a bit of back-and-forth during the review process. 

## Expected Behaviors

Before getting into the implementation details, there are some behaviors that are considered "core functionality" for Antidote, and it is the responsibility of all backends to ensure they're fully supported. These items should not depend on which backend happens to be running behind the scenes.

1. DNS lookups by endpoint name
2. `/antidote` (or otherwise configured) directory mapping
3. All presentation types, using the conventions established (e.g. the subdomains for HTTP endpoints will need L7 routing)

## Implementation Type and Directory Structure

Each new backend should have it's own directory within `scheduler/backends/` As an example, the kubernetes backend is located at `scheduler/backends/kubernetes.go`. This directory will contain a single package which is similarly named.

Each backend should have a high-level struct named according to the infrastructure it integrates with, followed by `Backend`. For instance, the Kubernetes backend is called `KubernetesBackend`. This struct should be located within a file named according to the name of the package and backend. For the kubernetes backend, this is `kubernetes.go`. This struct can have whatever fields are needed for the backend to maintain its own state, but must include the three fields shown in the below example:

```go
type KubernetesBackend struct {
	Config    config.AntidoteConfig
	Db        db.DataManager
	BuildInfo map[string]string

    // other fields specific to this backend can follow here.
}
```

Backend-specific struct fields can be used to maintain some state specific to that backend, (like a handle on an API client, etc), however the `DataManager` passed to the backend by the schedule should be used for **all** Antidote-specific state like `livelessons`, `livesessions`, etc. It is highly encouraged that contributors review the package in `db/` first, as this is where state management functions are defined, and their use is required.

There must also be a factory function in the same file that returns an initialized instance of this type, ready to be used by the scheduler. All setup activities should take place here, and this function should receive types that will satisfy the field requirements of the struct defined above. See `NewKubernetesBackend` for an example of this.

## The "AntidoteBackend" Interface

The Antidote scheduler interacts with a backend using a small handful of high-level functions. These functions are defined in a Go interface called [`AntidoteBackend`](https://github.com/nre-learning/antidote-core/blob/master/scheduler/scheduler.go). 

This interface is intentionally kept light, as there are only a handful of high-level functions that all backends must implement in order for Antidote to work with them. However, satisfying this interface is non-optional. The struct described in the previous section must fully implement `AntidoteBackend`.

In addition to satisfying this interface, there are several implementation details that must be adhered to. These details are documented as a docstring above each function reference in the `AntidoteBackend` interface. so please read those comments. There are also implementation details that apply to ALL functions in this interface, which are captured as a docstring above the interface. You should read all these comments before working on a backend.

## Additional Implementation Details

- All functions should be instrumented via OpenTracing - not just functions implementing the `AntidoteBackend` interface. See the Kubernetes backend for several examples.
- Network traffic should be constrained to the smallest possible unit to allow freely flowing intra-lesson communication. Network activity destined outside this boundary should be prohibited by default. This behavior should be disabled in the event that the AllowEgress configuration option is set to true.
- `antidote-images`, while opinionated towards Kubernetes in their current form, are a necessary component to antidote. There is a need to have platform-versioned images for any back-end infrastructure that `antidoted` can just use to run configuration scripts, jupyter notebooks, etc. It is highly recommended that any backend development effort augment that repository to add build scripts for each of those images relevant to that backend's infrastructure target, and then using those images in the same way that the kubernetes backend does currently.
- It's likely that additional backends will require this anyways, but support for private image sources is not optional. There should at least be an option to supply credentials when accessing an endpoint image to be used by a backend.
- The [`LiveLesson` data model](https://github.com/nre-learning/antidote-core/blob/master/db/models/livelesson.go) and its sub-fields provide a lot of useful information for creating resources on the backend side. For instance, HTTP presentations should use the `HepDomain` property of the `LivePresentation` model, as this is populated automatically by the API on creation. Similarly, if jupyter notebooks are used, the `GuideDomain` property of `LiveLesson` is similarly pre-populated, and kept up to date with stage changes. Developers should review this data model to be familiar with what it contains.

## Tests

Backend implementations should come with reasonably thorough unit tests. Important areas of focus for testing include but are not limited to:

- Garbage collection for lesson resources (and livelesson state) are properly cleaned up when appropriate
- Infrastructure creation functions handle all (within reason) failure scenarios properly
- Infrastructure creation functions handle all Antidote state updates properly

## Patience

The ability to have multiple backends in Antidote is new, and likely any subsequent backend implementation is sure to find some gaps in this documentation that will come out during the review process. Please be flexible and ready to adapt during the course of an implementation.
