generate: {
  type: genrule/copy
  in: "test.in"
  out: "test.h"
}

main: {
  type: c++/binary
  srcs: [
    "main.cc"
  ]

  deps: [
    ":generate"
  ]
}
