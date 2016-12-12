# JBuild
JBuild is a cross-platform, C++ build system written in Go. JBuild aims to have
a very simple configuration compared to other build systems (such as CMake) to
allow small projects to get up and running quickly, while still being scalable
and fast.

Many of the ideas in JBuild are inspired by [bazel](https://www.bazel.io/). This
project was started because it's fun to write and I wanted something which
worked natively on both Windows and Linux.

## Installation
JBuild can currently be installed using Go's awesome toolset:

`
go get -u github.com/jeshuam/jbuild && go install github.com/jeshuam/jbuild
`

## Quick Example
Let's say you had a library (containing some source and header files),
and a binary which uses that library. That's super simple in JBuild.

You could define a BUILD file like this:

```
main: {
  type: c++/binary
  srcs: ["main.cc"]
  deps: [":lib"]
}

lib: {
  type: c++/library
  srcs: ["lib.cc"]
  hdrs: ["lib.h"]
}
```

... and then run this:

```
jbuild run :main
```

This will compile `lib` and link it into a binary called `main`. It
will then run the final executable.

Want more? Check out the [Wiki](https://github.com/jeshuam/jbuild/wiki) for more examples/a comprehensive list
of supported features.

## Testing
### Unit Tests
To run unit tests, use something like:

```
go test github.com/jeshuam/jbuild/...
```

### Functional Tests
To run functional tests, use something like:

```
go test github.com/jeshuam/jbuild
```

To run functional tests with coverage, use:

```
go test -coverprofile=coverage.out -coverpkg=github.com/jeshuam/jbuild/... github.com/jeshuam/jbuild && go tool cover -html=coverage.out && rm coverage.out
```
