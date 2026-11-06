from conan import ConanFile


class Recipe(ConanFile):
    settings = "os", "compiler", "build_type", "arch"
    generators = "CMakeToolchain", "CMakeDeps", "VirtualRunEnv"

    def layout(self):
        self.folders.generators = "conan"

    def requirements(self):
        self.requires("fmt/11.0.2")
        self.requires("spdlog/1.15.0")
        self.requires("tomlplusplus/3.4.0")
        self.requires("libalsa/1.2.13")
        self.requires("opus/1.5.2")

    def build_requirements(self):
        self.test_requires("catch2/3.7.1")
