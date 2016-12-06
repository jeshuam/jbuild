#include <stdio.h>

#include <vector>

int main(int argc, char** argv) {
  std::vector<char> passed({'P', 'A', 'S', 'S', 'E', 'D'});
  for (const auto& c : passed) {
    printf("%c", c);
  }
}
