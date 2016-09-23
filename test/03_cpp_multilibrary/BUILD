hello_world: {
  type: c++/binary
  srcs: ["main.cc"]
  deps: [":lib", ":lib2"]
}

lib: {
  type: c++/library
  srcs: ["lib.cc"]
  deps: [":lib2"]
}

lib2: {
  type: c++/library
  srcs: ["lib2.cc"]
}
