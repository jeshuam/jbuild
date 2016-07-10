#include "lib/tool/tool.h"

#include <string>
#include <stdio.h>

int main() {
  std::string h(hello());
  std::string w(world());

  if (h != "Hello") {
    printf("FAIL\n");
    return 1;
  }

  if (w != "World") {
    printf("FAIL\n");
    return 1;
  }

  printf("PASS\n");
  return 0;
}
