#include <catch2/catch_test_macros.hpp>

#include "lvgl_main/runtime_status_mailbox.hpp"

using namespace lvgl_main;

TEST_CASE("runtime status mailbox coalesces to the highest pending revision")
{
	RuntimeStatusMailbox mailbox;
	RuntimeStatus first_status;
	first_status.app_version = "first";
	RuntimeStatus second_status;
	second_status.app_version = "second";

	REQUIRE(mailbox.offer({ .status = first_status, .revision = 1 }));
	REQUIRE(mailbox.offer({ .status = second_status, .revision = 2 }));

	const auto publication = mailbox.take_next();
	REQUIRE(publication.has_value());
	CHECK(publication->revision == 2);
	CHECK(publication->status.app_version == "second");
}

TEST_CASE("runtime status mailbox rejects an older callback after newer apply")
{
	RuntimeStatusMailbox mailbox;
	RuntimeStatus newer_status;
	newer_status.app_version = "newer";
	RuntimeStatus older_status;
	older_status.app_version = "older";

	REQUIRE(mailbox.offer({ .status = newer_status, .revision = 2 }));
	const auto applied = mailbox.take_next();
	REQUIRE(applied.has_value());
	CHECK(applied->status.app_version == "newer");

	CHECK_FALSE(mailbox.offer({ .status = older_status, .revision = 1 }));
	CHECK_FALSE(mailbox.take_next().has_value());
}

TEST_CASE("runtime status mailbox closes before runtime teardown")
{
	RuntimeStatusMailbox mailbox;
	RuntimeStatus status;
	status.app_version = "pending";

	REQUIRE(mailbox.offer({ .status = status, .revision = 1 }));
	mailbox.close();

	CHECK_FALSE(mailbox.take_next().has_value());
	CHECK_FALSE(mailbox.offer({ .status = status, .revision = 2 }));
}
