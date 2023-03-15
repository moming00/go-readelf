#include <iostream>

// g++ -g -Wl,--build-id ./*.cpp -o hello
int main() {
    std::cout << "hello world!\n";
    return 0;
}

// g++ -g -Wl,--build-id ./*.cpp -o hello
// ../go-readelf hello
// cp hello hello.origin
// mv hello hello.stripped
// objcopy --only-keep-debug hello.stripped hello.debug
// objcopy --strip-debug hello.stripped 
// objcopy --add-gnu-debuglink=hello.debug hello.stripped 
// objdump -s -j .gnu_debuglink ./hello.stripped