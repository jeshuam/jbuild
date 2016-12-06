#include <stdio.h>

#include <gtest/gtest.h>

TEST(A, Test) {
  ASSERT_EQ(1, 1);
}

int main(int argc, char** argv) {
  printf("PASSED");
}
