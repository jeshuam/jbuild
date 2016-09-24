main_file: {
  type: filegroup
  files: ["main.cc"]
}

lib_file: {
  type: filegroup
  files: ["lib.cc"]
}

hello_world_files: {
  type: filegroup
  files: [":main_file", ":lib_file"]
}

hello_world: {
  type: c++/binary
  srcs: [":hello_world_files"]
}
