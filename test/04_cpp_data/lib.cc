#include <string>
#include <fstream>

std::string passed() {
  std::ifstream t("data.txt");
  std::string str((std::istreambuf_iterator<char>(t)),
                   std::istreambuf_iterator<char>());
  return str;
}
