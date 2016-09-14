#!/bin/bash

cat >test.h <<EOF
#pragma once

#include <stdio.h>

void helloworld() {
  printf("Hello, world1!\n");
}
EOF


cat >test2.h <<EOF
#pragma once

#include <stdio.h>

void helloworld2() {
  printf("Hello, world2!\n");
}
EOF
