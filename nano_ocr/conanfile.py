from conan import ConanFile


class Recipe(ConanFile):
    settings = "os", "compiler", "build_type", "arch"
    generators = "CMakeToolchain", "CMakeDeps", "VirtualRunEnv"

    def layout(self):
        self.folders.generators = "conan"

    def requirements(self):
        self.requires("libcurl/8.20.0")
        self.requires("nlohmann_json/3.11.3")

    def build_requirements(self):
        self.test_requires("catch2/3.7.1")
        self.tool_requires("cmake/3.31.6")
