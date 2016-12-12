#include <stdio.h>

#include "util/string/format.h"

int main(int argc, char** argv) {
  std::string s = string::FormatMap("{message}", {{"message", "PASSED"}});
  printf(s.c_str());
}
