#include <string>

#include <catch2/catch_test_macros.hpp>

#include "systemd_dbus_client.hpp"

using namespace lvgl_main::detail;

TEST_CASE("systemd D-Bus unit paths use systemd object-path escaping")
{
	CHECK(systemd_unit_object_path("ssh.service") ==
	      "/org/freedesktop/systemd1/unit/ssh_2eservice");
	CHECK(systemd_unit_object_path("ssh.socket") ==
	      "/org/freedesktop/systemd1/unit/ssh_2esocket");
	CHECK(systemd_unit_object_path("foo_bar@0.service") ==
	      "/org/freedesktop/systemd1/unit/foo_5fbar_400_2eservice");
}

TEST_CASE("systemd D-Bus unit state matches systemctl status semantics")
{
	CHECK(systemd_dbus_unit_is_active({"active", "loaded"}));
	CHECK(systemd_dbus_unit_is_active({"reloading", "loaded"}));
	CHECK_FALSE(systemd_dbus_unit_is_active({"activating", "loaded"}));
	CHECK_FALSE(systemd_dbus_unit_is_active({"inactive", "loaded"}));
	CHECK_FALSE(systemd_dbus_unit_is_active({"active", "not-found"}));

	CHECK(systemd_dbus_unit_exists({"inactive", "loaded"}));
	CHECK_FALSE(systemd_dbus_unit_exists({"inactive", "not-found"}));

	CHECK(systemd_dbus_unit_is_stopped({"inactive", "loaded"}));
	CHECK(systemd_dbus_unit_is_stopped({"failed", "loaded"}));
	CHECK_FALSE(systemd_dbus_unit_is_stopped({"deactivating", "loaded"}));
	CHECK_FALSE(systemd_dbus_unit_is_stopped({"inactive", "not-found"}));
}
