from conan import ConanFile


class Recipe(ConanFile):
    settings = "os", "compiler", "build_type", "arch"
    generators = "CMakeToolchain", "CMakeDeps", "VirtualRunEnv"

    def layout(self):
        self.folders.generators = "conan"

    def requirements(self):
        self.requires("fmt/11.0.2")
        self.requires("spdlog/1.15.0")
        self.requires("libunwind/1.8.1")
        self.requires("cxxopts/3.2.0")
        self.requires("tomlplusplus/3.4.0")
        self.requires("andreasbuhr-cppcoro/cci.20230629")
        self.requires("nlohmann_json/3.11.3")
        self.requires("cpp-httplib/0.39.0")

    def build_requirements(self):
        self.test_requires("catch2/3.7.1")
        self.tool_requires("cmake/3.31.6")
