"""
Test script for cache clearing endpoints.

This script demonstrates how to use the new cache clearing API endpoints.
Run this to verify the cache clearing functionality.
"""

import asyncio
import httpx
from datetime import datetime

SERVER_URL = "http://localhost:8000"


async def check_cache_status():
    """Check current cache status."""
    print("\n📊 Checking Cache Status...")
    print("=" * 60)

    async with httpx.AsyncClient() as client:
        try:
            response = await client.get(f"{SERVER_URL}/api/v1/health/cache")
            response.raise_for_status()
            data = response.json()

            print(f"Cache Enabled: {data.get('cache_enabled')}")
            print(f"Backend: {data.get('backend')}")
            print(f"TTL Config: {data.get('ttl_config')}")
            return True
        except Exception as e:
            print(f"❌ Error: {e}")
            return False


async def clear_all_cache():
    """Clear all cache from Upstash Redis."""
    print("\n🗑️  Clearing All Cache...")
    print("=" * 60)

    async with httpx.AsyncClient() as client:
        try:
            response = await client.post(
                f"{SERVER_URL}/api/v1/health/cache/clear", timeout=10.0
            )

            if response.status_code == 200:
                data = response.json()
                print(f"✅ Success: {data.get('message')}")
                print(f"Timestamp: {data.get('timestamp')}")
                return True
            elif response.status_code == 503:
                data = response.json()
                print(f"❌ Service Unavailable: {data.get('detail')}")
                return False
            else:
                print(f"❌ Error (Status {response.status_code}): {response.text}")
                return False
        except httpx.ConnectError:
            print(f"❌ Connection Error: Cannot connect to {SERVER_URL}")
            print("   Make sure vnstock-server is running")
            return False
        except Exception as e:
            print(f"❌ Error: {e}")
            return False


async def get_rate_limit_status():
    """Get rate limit status."""
    print("\n⏱️  Checking Rate Limit Status...")
    print("=" * 60)

    async with httpx.AsyncClient() as client:
        try:
            response = await client.get(f"{SERVER_URL}/api/v1/health/quota")
            response.raise_for_status()
            data = response.json()

            print(f"Tier: {data.get('tier')}")
            daily = data.get("daily_quota", {})
            print(f"Daily Quota: {daily.get('used')}/{daily.get('total')} used")
            print(f"Warning Level: {data.get('warning_level')}")
            return True
        except Exception as e:
            print(f"❌ Error: {e}")
            return False


async def main():
    """Run all cache management tests."""
    print("\n" + "=" * 60)
    print("VNStock Server - Cache Management API Test")
    print(f"Server: {SERVER_URL}")
    print(f"Time: {datetime.now().isoformat()}")
    print("=" * 60)

    # Check cache status
    status_ok = await check_cache_status()

    # Check rate limit
    rate_ok = await get_rate_limit_status()

    # Clear cache
    clear_ok = await clear_all_cache()

    # Summary
    print("\n" + "=" * 60)
    print("Test Summary")
    print("=" * 60)
    print(f"Cache Status Check: {'✅ PASS' if status_ok else '❌ FAIL'}")
    print(f"Rate Limit Check: {'✅ PASS' if rate_ok else '❌ FAIL'}")
    print(f"Cache Clearing: {'✅ PASS' if clear_ok else '❌ FAIL'}")

    if status_ok and clear_ok:
        print("\n✨ All tests passed! Cache API is working correctly.")
    else:
        print("\n⚠️  Some tests failed. Check the errors above.")


if __name__ == "__main__":
    print("\n📌 Note: Make sure Upstash Redis is configured in .env.local")
    asyncio.run(main())
