#!/bin/bash

if [ -z "$1" ]; then
    echo "Usage: $0 <project_name>" >&2
    exit 1
fi

readonly name="$1"

echo "Creating project structure for: $name"

mkdir -p "$name/include/$name"
mkdir -p "$name/src"
echo "✓ Created directory structure"

touch "$name/src/tmp.cpp"
echo "✓ Created tmp.cpp"

cat <<EOF >"$name/CMakeLists.txt"
add_library(
    ${name} OBJECT
    \${CMAKE_CURRENT_SOURCE_DIR}/src/tmp.cpp
)

target_include_directories(
    ${name} PUBLIC
    \$<BUILD_INTERFACE:\${CMAKE_CURRENT_SOURCE_DIR}/include>
)

target_link_libraries(
    ${name} PRIVATE
    interface
    global_para
    logger
    tools
)

target_compile_features(${name} PUBLIC cxx_std_20)
EOF

echo "✓ Generated CMakeLists.txt"
echo "Project setup complete in './$name' directory"
