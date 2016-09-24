lib: {
  type: c++/library
  hdrs: ["include/lib.h"]
  includes: ["include"]
}

hello_world: {
  type: c++/binary
  srcs: ["main.cc"]
  deps: [":lib"]
}
