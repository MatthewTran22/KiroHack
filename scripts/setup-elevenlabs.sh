#!/bin/bash

# Setup script for ElevenLabs STT/TTS integration
set -e

echo "🎤 ElevenLabs STT/TTS Setup"
echo "=========================="

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

print_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }

# Check if .env file exists
if [ ! -f ".env" ]; then
    print_warning ".env file not found, copying from .env.example"
    cp .env.example .env
fi

# Check if ElevenLabs API key is set
if grep -q "your-elevenlabs-api-key-here" .env; then
    print_warning "ElevenLabs API key not configured in .env file"
    echo ""
    echo "To get your ElevenLabs API key:"
    echo "1. Go to https://elevenlabs.io/"
    echo "2. Sign up or log in to your account"
    echo "3. Go to your Profile settings"
    echo "4. Copy your API key"
    echo "5. Replace 'your-elevenlabs-api-key-here' in .env with your actual API key"
    echo ""
    read -p "Do you want to enter your API key now? (y/n): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        read -p "Enter your ElevenLabs API key: " api_key
        if [ ! -z "$api_key" ]; then
            # Update .env file
            if [[ "$OSTYPE" == "darwin"* ]]; then
                # macOS
                sed -i '' "s/your-elevenlabs-api-key-here/$api_key/" .env
            else
                # Linux
                sed -i "s/your-elevenlabs-api-key-here/$api_key/" .env
            fi
            print_success "API key updated in .env file"
        fi
    fi
else
    print_success "ElevenLabs API key is configured"
fi

# Check if audio file exists for testing
if [ ! -f "Rachel_cxItChZ1GQgNcn8hZVqT.wav" ]; then
    print_warning "Test audio file 'Rachel_cxItChZ1GQgNcn8hZVqT.wav' not found"
    print_info "You'll need an audio file to test the STT functionality"
else
    print_success "Test audio file found"
fi

# Test ElevenLabs API connection
print_info "Testing ElevenLabs API connection..."

# Source the .env file to get the API key
if [ -f ".env" ]; then
    export $(grep -v '^#' .env | xargs)
fi

if [ -z "$ELEVENLABS_API_KEY" ] || [ "$ELEVENLABS_API_KEY" = "your-elevenlabs-api-key-here" ]; then
    print_warning "Cannot test API connection - API key not configured"
else
    # Test API connection
    response=$(curl -s -w "%{http_code}" -H "xi-api-key: $ELEVENLABS_API_KEY" \
        "https://api.elevenlabs.io/v1/user" -o /tmp/elevenlabs_test.json)
    
    if [ "$response" = "200" ]; then
        print_success "ElevenLabs API connection successful!"
        
        # Show user info if available
        if command -v jq &> /dev/null; then
            user_info=$(cat /tmp/elevenlabs_test.json | jq -r '.subscription.tier // "free"')
            print_info "Subscription tier: $user_info"
        fi
    else
        print_error "ElevenLabs API connection failed (HTTP $response)"
        if [ -f "/tmp/elevenlabs_test.json" ]; then
            print_info "Response: $(cat /tmp/elevenlabs_test.json)"
        fi
    fi
    
    # Cleanup
    rm -f /tmp/elevenlabs_test.json
fi

print_info "Setup completed!"
echo ""
print_info "Next steps:"
echo "1. Make sure your ElevenLabs API key is configured in .env"
echo "2. Start the microservice: docker-compose --profile speech up -d elevenlabs-service"
echo "3. Test STT functionality with the API endpoints"
echo "4. Start the full application: make dev"
echo ""
print_info "ElevenLabs STT/TTS Features:"
echo "✅ Cloud-based processing (no local models needed)"
echo "✅ High-quality speech synthesis"
echo "✅ Accurate speech recognition"
echo "✅ Multiple language support"
echo "✅ Fast processing times"
echo "✅ No Docker complexity"