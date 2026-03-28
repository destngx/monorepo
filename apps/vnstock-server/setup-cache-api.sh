#!/bin/bash
# Quick start guide for using the cache clearing API

set -e

echo "🔧 VNStock Server - Cache Clearing Setup"
echo "=========================================="
echo ""

# Check if .env.local exists
if [ ! -f ".env.local" ]; then
    echo "📝 Creating .env.local from .env.example..."
    cp .env.example .env.local
    echo "✅ .env.local created"
    echo ""
    echo "⚠️  Please configure the following in .env.local:"
    echo "   - VNSTOCK_API_KEY"
    echo "   - UPSTASH_REDIS_REST_URL"
    echo "   - UPSTASH_REDIS_REST_TOKEN"
    echo "   - CACHE_ADMIN_API_KEY (new)"
    echo ""
else
    echo "✅ .env.local already exists"
fi

# Check if CACHE_ADMIN_API_KEY is set
if ! grep -q "CACHE_ADMIN_API_KEY" .env.local; then
    echo "⚠️  CACHE_ADMIN_API_KEY not found in .env.local"
    echo "   Adding it now..."
    echo "" >> .env.local
    echo "# Admin API Key for cache management" >> .env.local
    echo "CACHE_ADMIN_API_KEY=your_secure_admin_key_here" >> .env.local
    echo "✅ Added CACHE_ADMIN_API_KEY placeholder"
fi

echo ""
echo "🚀 Next steps:"
echo "   1. Update CACHE_ADMIN_API_KEY in .env.local with a secure key"
echo "   2. Start the server: python -m src.main"
echo "   3. Test cache clearing: python test_cache_api.py"
echo ""
echo "📖 For more details, see CACHE_MANAGEMENT.md"
