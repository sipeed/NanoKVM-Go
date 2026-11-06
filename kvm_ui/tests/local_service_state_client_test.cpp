#include <nlohmann/json.hpp>

#include <catch2/catch_test_macros.hpp>

#include "ui/local_service_state_client.hpp"

using namespace ui::local_service_state;

TEST_CASE("local service client parses MCP and screen timelapse state")
{
	const auto result = parse_services_response(R"json({
        "revision": 7,
        "services": {
            "mcp": {"enabled": true, "supported": true},
            "memoryFabric": {"enabled": false, "supported": false, "running": false},
            "screenTimelapse": {"enabled": false, "supported": true}
        }
    })json");

	REQUIRE(result);
	CHECK(result.services.revision == 7);
	CHECK(result.services.mcp_enabled);
	CHECK(result.services.mcp_supported);
	CHECK_FALSE(result.services.memory_fabric_enabled);
	CHECK_FALSE(result.services.memory_fabric_supported);
	CHECK(result.services.memory_fabric_running_known);
	CHECK_FALSE(result.services.memory_fabric_running);
	CHECK_FALSE(result.services.screen_timelapse_enabled);
	CHECK(result.services.screen_timelapse_supported);
}

TEST_CASE("local service client parses memory fabric and enabled timelapse")
{
	const auto result = parse_services_response(R"json({
        "revision": 8,
        "services": {
            "mcp": {"enabled": false, "supported": true},
            "memoryFabric": {"enabled": true, "supported": true, "running": true},
            "screenTimelapse": {"enabled": true, "supported": true}
        }
    })json");

	REQUIRE(result);
	CHECK_FALSE(result.services.mcp_enabled);
	CHECK(result.services.mcp_supported);
	CHECK(result.services.memory_fabric_enabled);
	CHECK(result.services.memory_fabric_supported);
	CHECK(result.services.memory_fabric_running_known);
	CHECK(result.services.memory_fabric_running);
	CHECK(result.services.screen_timelapse_enabled);
	CHECK(result.services.screen_timelapse_supported);
}

TEST_CASE("local service client rejects malformed or incomplete JSON")
{
	CHECK_FALSE(parse_services_response("{"));
	CHECK_FALSE(parse_services_response(R"json({"services":{}})json"));
	CHECK_FALSE(parse_services_response(R"json({
        "services": {
            "mcp": {"enabled": true},
            "memoryFabric": {"enabled": false},
            "screenTimelapse": {"enabled": "yes"}
        }
    })json"));
}

TEST_CASE("MCP bundle active follows enabled state independent of support")
{
	ServicesSnapshot services;
	CHECK_FALSE(mcp_bundle_active(services));

	services.mcp_enabled = true;
	CHECK(mcp_bundle_active(services));

	services.mcp_enabled = false;
	services.mcp_supported = true;
	CHECK_FALSE(mcp_bundle_active(services));

	services.memory_fabric_enabled = true;
	CHECK(mcp_bundle_active(services));

	services.memory_fabric_supported = true;
	CHECK(mcp_bundle_active(services));

	services.memory_fabric_enabled = false;
	CHECK_FALSE(mcp_bundle_active(services));

	services.mcp_enabled = true;
	CHECK(mcp_bundle_active(services));

	services.mcp_enabled = false;
	services.memory_fabric_enabled = true;
	CHECK(mcp_bundle_active(services));

	services.mcp_enabled = true;
	CHECK(mcp_bundle_active(services));
}

TEST_CASE("screen timelapse active follows enabled state independent of support")
{
	ServicesSnapshot services;
	CHECK_FALSE(screen_timelapse_active(services));

	services.screen_timelapse_enabled = true;
	CHECK(screen_timelapse_active(services));

	services.screen_timelapse_supported = true;
	CHECK(screen_timelapse_active(services));

	services.screen_timelapse_enabled = false;
	CHECK_FALSE(screen_timelapse_active(services));
}

TEST_CASE("MCP bundle request order is support aware")
{
	const auto mcp_only_order = mcp_bundle_service_order(false);
	REQUIRE(mcp_only_order.size() == 1);
	CHECK(mcp_only_order[0] == kMCPServiceID);

	const auto bundle_order = mcp_bundle_service_order(true);
	REQUIRE(bundle_order.size() == 2);
	CHECK(bundle_order[0] == kMCPServiceID);
	CHECK(bundle_order[1] == kMemoryFabricServiceID);
}

TEST_CASE("local service client builds screen source PUT body")
{
	const auto enabled =
		nlohmann::json::parse(build_set_service_request_body(true));
	CHECK(enabled.at("enabled").get<bool>());
	CHECK(enabled.at("source").get<std::string>() == "screen");

	const auto disabled =
		nlohmann::json::parse(build_set_service_request_body(false));
	CHECK_FALSE(disabled.at("enabled").get<bool>());
	CHECK(disabled.at("source").get<std::string>() == "screen");
}

TEST_CASE("local service client parses HTTP response status")
{
	const auto ok = detail::parse_http_response(
		"HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n\r\n{}");
	REQUIRE(ok.ok);
	CHECK(ok.status_code == 200);
	CHECK(ok.body == "{}");

	const auto failed = detail::parse_http_response(
		"HTTP/1.1 500 Internal Server Error\r\nContent-Type: application/json\r\n\r\n{}");
	CHECK_FALSE(failed.ok);
	CHECK(failed.status_code == 500);
}
