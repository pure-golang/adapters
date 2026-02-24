#!/bin/bash
set -e

# FCM Setup Script - Universal setup from scratch
# Automatically configures GCP project, creates service account,
# sets up Firebase, and prepares for testing

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Default values
DEFAULT_SERVICE_ACCOUNT="fcm-sender"
DEFAULT_CREDENTIALS_PATH="./credentials.json"
PROJECT_ID=""

# Functions
print_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1" >&2; }
print_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
print_header() {
    echo ""
    echo -e "${BLUE}============================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}============================================${NC}"
    echo ""
}

# Check prerequisites
check_prerequisites() {
    print_header "Checking Prerequisites"

    if ! command -v gcloud &> /dev/null; then
        print_error "gcloud CLI not installed"
        echo "Install: brew install google-cloud-sdk"
        exit 1
    fi
    print_success "gcloud CLI installed"

    if ! gcloud auth list --filter="status:ACTIVE" --format="value(account)" 2>/dev/null | grep -q .; then
        print_error "Not authenticated"
        echo "Run: gcloud auth login"
        exit 1
    fi
    local auth_user=$(gcloud auth list --filter="status:ACTIVE" --format="value(account)")
    print_success "Authenticated as: $auth_user"

    if command -v jq &> /dev/null; then
        print_success "jq installed"
    else
        print_warning "jq not installed (optional)"
    fi

    if command -v go &> /dev/null; then
        print_success "Go installed"
    else
        print_warning "Go not installed (needed for tests)"
    fi
}

# Prompt for project ID
prompt_project_id() {
    print_header "Project Configuration"

    echo "Available projects:"
    gcloud projects list 2>/dev/null || echo "  (Could not list projects)"
    echo ""

    while true; do
        read -p "Enter your GCP Project ID: " PROJECT_ID

        if [ -z "$PROJECT_ID" ]; then
            print_error "Project ID cannot be empty"
            continue
        fi

        if gcloud projects describe "$PROJECT_ID" &>/dev/null; then
            print_success "Project '$PROJECT_ID' exists"
            break
        else
            print_error "Project not found"
        fi
    done
}

# Setup complete pipeline
setup_complete() {
    print_header "Setting Up FCM"

    local project_number=$(gcloud projects describe "$PROJECT_ID" --format='value(projectNumber)')
    print_info "Project Number: $project_number"

    # Enable APIs
    print_info "Enabling Firebase API..."
    gcloud services enable firebase.googleapis.com --project="$PROJECT_ID" --quiet
    print_success "Firebase API enabled"

    print_info "Enabling FCM API..."
    gcloud services enable fcm.googleapis.com --project="$PROJECT_ID" --quiet
    print_success "FCM API enabled"

    # Create service account
    local sa_email="$DEFAULT_SERVICE_ACCOUNT@$PROJECT_ID.iam.gserviceaccount.com"

    print_info "Setting up service account..."
    if gcloud iam service-accounts describe "$sa_email" --project="$PROJECT_ID" &>/dev/null; then
        print_warning "Service account already exists: $sa_email"
    else
        gcloud iam service-accounts create "$DEFAULT_SERVICE_ACCOUNT" \
            --display-name="FCM Service Account" \
            --project="$PROJECT_ID" --quiet
        print_success "Service account created: $sa_email"
    fi

    # Grant IAM role
    print_info "Granting Firebase Admin SDK role..."
    gcloud projects add-iam-policy-binding "$PROJECT_ID" \
        --member="serviceAccount:$sa_email" \
        --role="roles/firebase.sdkAdminServiceAgent" \
        --quiet 2>/dev/null || print_warning "IAM role may already be granted"

    # Create credentials
    print_info "Creating service account key..."
    gcloud iam service-accounts keys create "$DEFAULT_CREDENTIALS_PATH" \
        --iam-account="$sa_email" \
        --project="$PROJECT_ID" --quiet
    chmod 600 "$DEFAULT_CREDENTIALS_PATH"
    print_success "Credentials file created: $DEFAULT_CREDENTIALS_PATH"
    print_warning "Keep this file secure and never commit it to version control!"

    # Verify
    print_header "Verification"

    print_info "Checking service account..."
    if gcloud iam service-accounts describe "$sa_email" --project="$PROJECT_ID" &>/dev/null; then
        print_success "Service account exists"
    else
        print_error "Service account not found"
        return 1
    fi

    print_info "Checking enabled APIs..."
    local apis=$(gcloud services list --enabled --project="$PROJECT_ID" --format="value(name)")
    if echo "$apis" | grep -q "firebase.googleapis.com"; then
        print_success "Firebase API enabled"
    else
        print_error "Firebase API not enabled"
        return 1
    fi

    if echo "$apis" | grep -q "fcm.googleapis.com"; then
        print_success "FCM API enabled"
    else
        print_warning "FCM API not enabled (may not be required)"
    fi

    # Firebase integration guide
    print_header "Firebase Integration"

    print_warning "⚠️  Important: Project must be linked to Firebase"
    echo ""
    echo "Steps to complete setup:"
    echo ""
    echo "1. Link GCP project to Firebase:"
    echo "   a) Open https://console.firebase.google.com/"
    echo "   b) Click 'Add project'"
    echo "   c) Select '$PROJECT_ID' from the list"
    echo "   d) Continue setup"
    echo ""
    echo "2. Create a Web App:"
    echo "   a) Go to Project Settings → General → Your apps"
    echo "   b) Click 'Add app' → '</>' (Web icon)"
    echo "   c) Register app (skip hosting setup)"
    echo "   d) Copy the firebaseConfig object"
    echo ""
    echo "3. Update test configuration:"
    echo "   cd test"
    echo "   ./serve.sh"
    echo "   Open http://localhost:8080/web_push_test.html"
    echo "   Enter your API Key and App ID from Firebase Console"
    echo ""
    echo "4. Get FCM token and test:"
    echo "   - Request Permission → Allow"
    echo "   - Get FCM Token → Copy"
    echo "   - Run: go run send.go ../credentials.json"
    echo ""

    # Wait for user to complete Firebase setup
    read -p "Press Enter after completing Firebase setup (or skip with Ctrl+C)..."

    # Summary
    print_header "Setup Summary"

    echo "Project ID: $PROJECT_ID"
    echo "Service Account: $sa_email"
    echo "Credentials File: $DEFAULT_CREDENTIALS_PATH"
    echo "Project Number: $project_number"
    echo ""
    echo "Next steps:"
    echo "  1. Complete Firebase integration (see above)"
    echo "  2. Get FCM token from web page"
    echo "   3. Test with: go run test/send.go ../credentials.json"
    echo ""
    print_success "GCP setup completed!"
}

# Main execution
main() {
    print_header "FCM Setup Script"

    check_prerequisites
    prompt_project_id
    setup_complete
}

main "$@"
