# libkvm

This is the libkvm project.

## build libkvm.so

Note: Ensure that the cross-compiler is installed. If not installed, please follow the instructions in [README.md](../README.md) to install it.

### Install conan

```shell
pip install conan
```

### Build

```shell
./builder.sh build               # build all (default build ax630)

./builder.sh build -p ax630      # build ax630

./builder.sh build -p h618       # build h618

# Output
ls build/libkvm.so*
```

Also you can clean or rebuild

```shell
./builder.sh clean      # clean target
./builder.sh rebuild    # clean && build
```
