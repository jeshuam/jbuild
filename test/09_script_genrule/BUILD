generate: {
  type: genrule/script
  in: "gen_test.h.sh"
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
