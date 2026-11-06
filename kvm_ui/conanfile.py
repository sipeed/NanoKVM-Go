from conan import ConanFile


class Recipe(ConanFile):
    settings = "os", "compiler", "build_type", "arch"
    generators = "CMakeToolchain", "CMakeDeps", "VirtualRunEnv"
    default_options = {
        "libcurl/*:shared": False,
        "libcurl/*:fPIC": True,
        "libcurl/*:build_executable": False,
        "libcurl/*:with_ssl": "openssl",
        "libcurl/*:with_file": False,
        "libcurl/*:with_ftp": False,
        "libcurl/*:with_http": True,
        "libcurl/*:with_ldap": False,
        "libcurl/*:with_rtsp": False,
        "libcurl/*:with_dict": False,
        "libcurl/*:with_telnet": False,
        "libcurl/*:with_tftp": False,
        "libcurl/*:with_pop3": False,
        "libcurl/*:with_imap": False,
        "libcurl/*:with_smb": False,
        "libcurl/*:with_smtp": False,
        "libcurl/*:with_gopher": False,
        "libcurl/*:with_mqtt": False,
        "libcurl/*:with_libssh2": False,
        "libcurl/*:with_libidn": False,
        "libcurl/*:with_libgsasl": False,
        "libcurl/*:with_libpsl": False,
        "libcurl/*:with_largemaxwritesize": False,
        "libcurl/*:with_nghttp2": False,
        "libcurl/*:with_zlib": False,
        "libcurl/*:with_brotli": False,
        "libcurl/*:with_zstd": False,
        "libcurl/*:with_c_ares": False,
        "libcurl/*:with_threaded_resolver": True,
        "libcurl/*:with_proxy": True,
        "libcurl/*:with_crypto_auth": False,
        "libcurl/*:with_ntlm": False,
        "libcurl/*:with_cookies": False,
        "libcurl/*:with_ipv6": True,
        "libcurl/*:with_docs": False,
        "libcurl/*:with_misc_docs": False,
        "libcurl/*:with_verbose_debug": False,
        "libcurl/*:with_symbol_hiding": False,
        "libcurl/*:with_unix_sockets": False,
        "libcurl/*:with_verbose_strings": False,
        "libcurl/*:with_ca_bundle": "auto",
        "libcurl/*:with_ca_path": "auto",
        "libcurl/*:with_ca_fallback": False,
        "libcurl/*:with_form_api": False,
        "libcurl/*:with_websockets": False,
    }

    def layout(self):
        self.folders.generators = "conan"

    def requirements(self):
        self.requires("spdlog/1.15.0")
        self.requires("tomlplusplus/3.4.0")
        self.requires("cpp-httplib/0.39.0")
        self.requires("libcurl/8.21.0")
        self.requires("nlohmann_json/3.11.3")
        self.requires("stb/cci.20240531")

    def build_requirements(self):
        self.test_requires("catch2/3.7.1")
        self.tool_requires("cmake/3.31.6")
