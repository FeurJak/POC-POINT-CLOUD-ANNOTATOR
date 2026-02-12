# Point Cloud Annotator

## Objective

The objective of this assignment is to build a simple web application that allows a user to load a 3D point cloud, annotate specific points, and have those annotations persist across page loads.

## Technical Requirements:

### Front-End:

The front-end should be built on top of "Potree" which is a WebGL point cloud viewer for large datasets. This fork is availble locally within the .tmp folder of this repository.

The Front-End should provide these additional user-interface functionalities:

- Create: A user must be able to click on any point in the 3D scene. This action should
  create an annotation "marker" at that 3D coordinate.
- Data: When an annotation is created, the user should be able to attach a simple
  string to it (max 256 bytes). This could be done via a simple text input box that
  appears.
- Delete: A user must be able to delete existing annotations.

A good point of reference for the annotation functionality is the Potree "annotation" example (.tmp/potree/examples/annotations.html)

### Back-End:

The backend is comprised of the following services:

- API-Gateway: This service will handle all incoming requests and route them to the appropriate hanlders (services).
- Database: This service will store all the annotations and their associated data.
- Handlers: These are services will handle all the business logic related to annotations.

#### API-Gateway:

The API-Gateway is a service implemented in Golang that handles all incoming requests and routes them to the appropriate handlers (services).

Handlers services are expected to be contained in an isolated network, thus inaccessible to to the outside world. The API-Gateway will be the only point of entry for all requests from the front-end.

The API-Gateway should be built with a dependency injection based application framework such as Uber [Fx](https://github.com/uber-go/fx.git).

#### Database:

Persistent storage will be available via a Postgres instance & data-cache will be available via a Redis instance. The API-Gateway & Handlers should take advantage of the Redis cache to improve performance and reduce database load, ensuring that the application remains responsive and scalable.

## Development Guidelines:

The latest up-to date best-practices should be followed when developing the golang services and it is recommended that the Uber Go Style Guide to be the Style Guide for development.

Ensure sufficient test-coverage with all Go implementations to guarantee code quality and maintainability. Interfaces should be well-defined and documented and mocks should be used to isolate dependencies for unit-tests.

## Deployment:

The API-Gateway, Database & Handlers should be contained within a Docker Compose file. This will allow for easy deployment and scaling of the application.

Connection to the API-Gateway can be specified either through a static default URI (for local-development) or through environment variables (for production).
