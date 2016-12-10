hello_world: {
  type: c++/binary
  srcs: [
    "main.cc",
    ":gen_pass",
    ":gen_ed",
  ]
}

fg: {
  type: filegroup
  files: ["printer.cc.in"]
}

gen_pass: {
  type: genrule
  in: [":fg"]
  out: ["pa.cc", "ss.cc"]
  cmds: [
    "sed 's/@WORD@/PA/' printer.cc.in > pa.cc",
    "sed 's/@WORD@/SS/' printer.cc.in > ss.cc",
  ]
}

gen_ed: {
  type: genrule
  in: ["printer.cc.in"]
  out: ["ed.cc"]
  cmds: [
    "sed 's/@WORD@/ED/' printer.cc.in > ed.cc",
  ]
}
