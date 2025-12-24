#!/bin/bash
# This script manages starting/stopping Ollama and docker-compose services as needed

# Set colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # no color

# Get the project root directory (parent of the script directory)
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# Version information
VERSION="1.0.1" # update this when changing the script
SCRIPT_NAME=$(basename "$0")

# Display help information
show_help() {
    printf "%b\n" "${GREEN}WeKnora start script v${VERSION}${NC}"
    printf "%b\n" "${GREEN}Usage:${NC} $0 [options]"
    echo "Options:"
    echo "  -h, --help     Show this help information"
    echo "  -o, --ollama   Start the Ollama service"
    echo "  -d, --docker   Start Docker container services"
    echo "  -a, --all      Start all services (default)"
    echo "  -s, --stop     Stop all services"
    echo "  -c, --check    Check the environment and diagnose issues"
    echo "  -r, --restart  Rebuild and restart a specific container"
    echo "  -l, --list     List all running containers"
    echo "  -p, --pull     Pull the latest Docker images"
    echo "  --no-pull      Skip pulling images when starting (default pulls)"
    echo "  -v, --version  Show version information"
    exit 0
}

# Display version information
show_version() {
    printf "%b\n" "${GREEN}WeKnora start script v${VERSION}${NC}"
    exit 0
}

# Log helper functions
log_info() {
    printf "%b\n" "${BLUE}[INFO]${NC} $1"
}

log_warning() {
    printf "%b\n" "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    printf "%b\n" "${RED}[ERROR]${NC} $1"
}

log_success() {
    printf "%b\n" "${GREEN}[SUCCESS]${NC} $1"
}

# Select the available Docker Compose command (prefer 'docker compose', fallback to 'docker-compose')
DOCKER_COMPOSE_BIN=""
DOCKER_COMPOSE_SUBCMD=""

detect_compose_cmd() {
# Prefer the Docker Compose plugin
	if docker compose version &> /dev/null; then
		DOCKER_COMPOSE_BIN="docker"
		DOCKER_COMPOSE_SUBCMD="compose"
		return 0
	fi

# Fallback to docker-compose (v1)
	if command -v docker-compose &> /dev/null; then
		if docker-compose version &> /dev/null; then
			DOCKER_COMPOSE_BIN="docker-compose"
			DOCKER_COMPOSE_SUBCMD=""
			return 0
		fi
	fi

# None are available
	return 1
}

# Check and create .env file if missing
check_env_file() {
    log_info "Checking environment variable configuration..."
    if [ ! -f "$PROJECT_ROOT/.env" ]; then
        log_warning ".env file missing; creating from template"
        if [ -f "$PROJECT_ROOT/.env.example" ]; then
            cp "$PROJECT_ROOT/.env.example" "$PROJECT_ROOT/.env"
            log_success "Created .env file from .env.example"
        else
            log_error "Could not find .env.example template; cannot create .env"
            return 1
        fi
    else
        log_info ".env file already exists"
    fi
    
# Check if required environment variables are set
    source "$PROJECT_ROOT/.env"
    local missing_vars=()
    
# Verify basic vars
    if [ -z "$DB_DRIVER" ]; then missing_vars+=("DB_DRIVER"); fi
    if [ -z "$STORAGE_TYPE" ]; then missing_vars+=("STORAGE_TYPE"); fi
    
    return 0
}

# Install Ollama (differs based on platform)
install_ollama() {
# Check if the configured service is remote
    get_ollama_base_url
    
    if [ $IS_REMOTE -eq 1 ]; then
        log_info "Remote Ollama configured; skipping local installation"
        return 0
    fi

    log_info "Local Ollama not installed; installing..."
    
    OS=$(uname)
    if [ "$OS" = "Darwin" ]; then
# macOS installation path
        log_info "macOS detected; installing Ollama via Homebrew..."
        if ! command -v brew &> /dev/null; then
# Download binary when Homebrew is not available
            log_info "Homebrew missing; downloading Ollama package..."
            curl -fsSL https://ollama.com/download/Ollama-darwin.zip -o ollama.zip
            unzip ollama.zip
            mv ollama /usr/local/bin
            rm ollama.zip
        else
            brew install ollama
        fi
    else
# Linux installation path
        log_info "Linux detected; running the install script..."
        curl -fsSL https://ollama.com/install.sh | sh
    fi
    
    if [ $? -eq 0 ]; then
        log_success "Local Ollama installation completed"
        return 0
    else
        log_error "Local Ollama installation failed"
        return 1
    fi
}

# Retrieve the Ollama base URL and determine if it is remote
get_ollama_base_url() {

    check_env_file

    # Read the Ollama base URL from env vars
    OLLAMA_URL=${OLLAMA_BASE_URL:-"http://host.docker.internal:11434"}
    # Extract host portion
    OLLAMA_HOST=$(echo "$OLLAMA_URL" | sed -E 's|^https?://||' | sed -E 's|:[0-9]+$||' | sed -E 's|/.*$||')
    # Extract port portion
    OLLAMA_PORT=$(echo "$OLLAMA_URL" | grep -oE ':[0-9]+' | grep -oE '[0-9]+' || echo "11434")
    # Determine if the service points to a local address
    IS_REMOTE=0
    if [ "$OLLAMA_HOST" = "localhost" ] || [ "$OLLAMA_HOST" = "127.0.0.1" ] || [ "$OLLAMA_HOST" = "host.docker.internal" ]; then
        IS_REMOTE=0  # local service
    else
        IS_REMOTE=1  # remote service
    fi
}

# Start Ollama service
start_ollama() {
    log_info "Checking Ollama service..."
    # Extract host and port
    get_ollama_base_url
    log_info "Ollama service URL: $OLLAMA_URL"
    
    if [ $IS_REMOTE -eq 1 ]; then
        log_info "Remote Ollama service detected; using remote endpoint without local install"
        # Check whether the remote service is reachable
        if curl -s "$OLLAMA_URL/api/tags" &> /dev/null; then
            log_success "Remote Ollama service is reachable"
            return 0
        else
            log_warning "Remote Ollama service is unreachable; please verify the endpoint and that it is running"
            return 1
        fi
    fi
    
    # Handle the local service
    # Check whether Ollama is installed locally
    if ! command -v ollama &> /dev/null; then
        install_ollama
        if [ $? -ne 0 ]; then
            return 1
        fi
    fi

    # Check if the service is already running
    if curl -s "http://localhost:$OLLAMA_PORT/api/tags" &> /dev/null; then
        log_success "Local Ollama service already running on port $OLLAMA_PORT"
    else
        log_info "Starting local Ollama service..."
        # Note: the official recommendation is to manage the service via systemctl or launchctl; backgrounding is for temporary scenarios
        systemctl restart ollama || (ollama serve > /dev/null 2>&1 < /dev/null &)
        
        # Wait for the service to start
        MAX_RETRIES=30
        COUNT=0
        while [ $COUNT -lt $MAX_RETRIES ]; do
            if curl -s "http://localhost:$OLLAMA_PORT/api/tags" &> /dev/null; then
                log_success "Local Ollama service started on port $OLLAMA_PORT"
                break
            fi
            echo -ne "Waiting for Ollama to start... ($COUNT/$MAX_RETRIES)\r"
            sleep 1
            COUNT=$((COUNT + 1))
        done
        echo "" # newline
        
        if [ $COUNT -eq $MAX_RETRIES ]; then
        log_error "Failed to start the local Ollama service"
            return 1
        fi
    fi

    log_success "Local Ollama service URL: http://localhost:$OLLAMA_PORT"
    return 0
}

# Stop the Ollama service
stop_ollama() {
    log_info "Stopping Ollama service..."
    
# Check if the service is remote
    get_ollama_base_url
    
    if [ $IS_REMOTE -eq 1 ]; then
        log_info "Remote Ollama detected; no local shutdown required"
        return 0
    fi
    
# Check if Ollama is installed locally
    if ! command -v ollama &> /dev/null; then
        log_info "Local Ollama not installed; nothing to stop"
        return 0
    fi
    
# Find and terminate any Ollama process
    if pgrep -x "ollama" > /dev/null; then
# Prefer using systemctl
        if command -v systemctl &> /dev/null; then
            sudo systemctl stop ollama
        else
            pkill -f "ollama serve"
        fi
        log_success "Local Ollama service stopped"
    else
        log_info "Local Ollama service not running"
    fi
    
    return 0
}

# Check if Docker is installed
check_docker() {
    log_info "Checking Docker environment..."
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed; please install Docker"
        return 1
    fi
    
# Ensure Docker Compose command is available
	if detect_compose_cmd; then
		if [ "$DOCKER_COMPOSE_BIN" = "docker" ]; then
			log_info "Detected Docker Compose plugin (docker compose)"
		else
			log_info "Detected docker-compose (v1)"
		fi
	else
		log_error "Neither 'docker compose' nor 'docker-compose' was found; please install one of them."
		return 1
	fi
    
# Check Docker daemon status
    if ! docker info &> /dev/null; then
        log_error "Docker daemon is not running; please start Docker"
        return 1
    fi
    
    log_success "Docker environment check passed"
    return 0
}

check_platform() {
     # Detect the current system platform
    log_info "Detecting system platform..."
    if [ "$(uname -m)" = "x86_64" ]; then
        export PLATFORM="linux/amd64"
    elif [ "$(uname -m)" = "aarch64" ] || [ "$(uname -m)" = "arm64" ]; then
        export PLATFORM="linux/arm64"
    else
        log_warning "Unrecognized platform: $(uname -m); defaulting to linux/amd64"
        export PLATFORM="linux/amd64"
    fi
    log_info "Current platform: $PLATFORM"
}

# Start Docker containers
start_docker() {
    log_info "Starting Docker containers..."
    
# Verify Docker environment
    check_docker
    if [ $? -ne 0 ]; then
        return 1
    fi
    
# Ensure .env file exists
    check_env_file
    
# Load .env file
    source "$PROJECT_ROOT/.env"
    storage_type=${STORAGE_TYPE:-local}
    
    check_platform
    
# Move into project root to run docker-compose commands
    cd "$PROJECT_ROOT"
    
    # Start core service containers
    log_info "Starting core service containers..."
	# Use the detected Compose command
	if [ "$NO_PULL" = true ]; then
		# Do not pull images, use existing local ones
		log_info "Skipping image pull; using local images..."
		PLATFORM=$PLATFORM "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD up --build -d
	else
		# Pull the latest images
		log_info "Pulling the latest images..."
		PLATFORM=$PLATFORM "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD up --pull always -d
	fi
    if [ $? -ne 0 ]; then
        log_error "Failed to start Docker containers"
        return 1
    fi
    
    log_success "All Docker containers started successfully"
    
    # Show container status
    log_info "Current container status:"
	"$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD ps
    
    return 0
}

# Stop Docker containers
stop_docker() {
    log_info "Stopping Docker containers..."
    
# Verify Docker environment
    check_docker
    if [ $? -ne 0 ]; then
        # Attempt to stop containers even if the check failed
        log_warning "Docker environment check failed; still attempting to stop containers..."
    fi
    
# Move into project root to run docker-compose commands
    cd "$PROJECT_ROOT"
    
# Stop all containers
	"$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD down --remove-orphans
    if [ $? -ne 0 ]; then
        log_error "Failed to stop Docker containers"
        return 1
    fi
    
    log_success "All Docker containers stopped"
    return 0
}

# List currently running containers
list_containers() {
    log_info "Listing running containers..."
    
    # Verify Docker environment
    check_docker
    if [ $? -ne 0 ]; then
        return 1
    fi
    
# Enter project root before invoking docker-compose
    cd "$PROJECT_ROOT"
    
    # Display running containers
    printf "%b\n" "${BLUE}Currently running containers:${NC}"
	"$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD ps --services | sort
    
    return 0
}

# Pull the latest Docker images
pull_images() {
    log_info "Pulling the latest Docker images..."
    
# Verify Docker environment
    check_docker
    if [ $? -ne 0 ]; then
        return 1
    fi
    
# Ensure .env file exists
    check_env_file
    
# Load .env file
    source "$PROJECT_ROOT/.env"
    storage_type=${STORAGE_TYPE:-local}
    
    check_platform
    
# Enter project root to run docker-compose
    cd "$PROJECT_ROOT"
    
    # Pull images for all services
    log_info "Pulling the latest images for all services..."
	PLATFORM=$PLATFORM "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD pull
    if [ $? -ne 0 ]; then
        log_error "Failed to pull images"
        return 1
    fi
    
    log_success "All images are now up to date"
    
    # Show pulled image info
    log_info "Pulled images:"
    docker images --format "table {{.Repository}}\t{{.Tag}}\t{{.CreatedAt}}\t{{.Size}}" | head -10
    
    return 0
}

# Restart a specific container
restart_container() {
    local container_name="$1"
    
    if [ -z "$container_name" ]; then
        log_error "No container name specified"
        echo "Available containers:"
        list_containers
        return 1
    fi
    
    log_info "Rebuilding and restarting container: $container_name"
    
# Verify Docker environment
    check_docker
    if [ $? -ne 0 ]; then
        return 1
    fi
    
    check_platform
    
# Change to the project root before running docker-compose
    cd "$PROJECT_ROOT"
    
    # Check if the container exists
	if ! "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD ps --services | grep -q "^$container_name$"; then
        log_error "Container '$container_name' does not exist or is not running"
        echo "Available containers:"
        list_containers
        return 1
    fi
    
    # Build and restart the container
    log_info "Rebuilding container '$container_name'..."
	PLATFORM=$PLATFORM "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD build "$container_name"
    if [ $? -ne 0 ]; then
        log_error "Failed to build container '$container_name'"
        return 1
    fi
    
    log_info "Restarting container '$container_name'..."
	PLATFORM=$PLATFORM "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD up -d --no-deps "$container_name"
    if [ $? -ne 0 ]; then
        log_error "Failed to restart container '$container_name'"
        return 1
    fi
    
    log_success "Container '$container_name' rebuilt and restarted successfully"
    return 0
}

check_environment() {
    log_info "Starting environment check..."
    
    # Check operating system
    OS=$(uname)
    log_info "Operating system: $OS"
    
    # Check Docker
    check_docker
    
# Verify .env file
    check_env_file
    
    get_ollama_base_url
    
    if [ $IS_REMOTE -eq 1 ]; then
        log_info "Remote Ollama configuration detected"
        if curl -s "$OLLAMA_URL/api/tags" &> /dev/null; then
            version=$(curl -s "$OLLAMA_URL/api/tags" | grep -o '"version":"[^"]*"' | cut -d'"' -f4)
            log_success "Remote Ollama reachable, version: $version"
        else
            log_warning "Remote Ollama unreachable; please verify the endpoint and ensure it is running"
        fi
    else
        if command -v ollama &> /dev/null; then
            log_success "Local Ollama is installed"
            if curl -s "http://localhost:$OLLAMA_PORT/api/tags" &> /dev/null; then
                version=$(curl -s "http://localhost:$OLLAMA_PORT/api/tags" | grep -o '"version":"[^"]*"' | cut -d'"' -f4)
                log_success "Local Ollama service is running, version: $version"
            else
                log_warning "Local Ollama installed but not running"
            fi
        else
            log_warning "Local Ollama is not installed"
        fi
    fi
    
    # Check disk space
    log_info "Checking disk space..."
    df -h | grep -E "(Filesystem|/$)"
    
    # Check memory
    log_info "Checking memory usage..."
    if [ "$OS" = "Darwin" ]; then
        vm_stat | perl -ne '/page size of (\d+)/ and $size=$1; /Pages free:\s*(\d+)/ and print "Free Memory: ", $1 * $size / 1048576, " MB\n"'
    else
        free -h | grep -E "(total|Mem:)"
    fi
    
    # Check CPU
    log_info "CPU information:"
    if [ "$OS" = "Darwin" ]; then
        sysctl -n machdep.cpu.brand_string
        echo "CPU cores: $(sysctl -n hw.ncpu)"
    else
        grep "model name" /proc/cpuinfo | head -1
        echo "CPU cores: $(nproc)"
    fi
    
    # Check container status
    log_info "Checking container status..."
    if docker info &> /dev/null; then
        docker ps -a
    else
        log_warning "Unable to fetch container status; Docker may not be running"
    fi
    
    log_success "Environment check complete"
    return 0
}

# Parse command-line arguments
START_OLLAMA=false
START_DOCKER=false
STOP_SERVICES=false
CHECK_ENVIRONMENT=false
LIST_CONTAINERS=false
RESTART_CONTAINER=false
PULL_IMAGES=false
NO_PULL=false
CONTAINER_NAME=""

# Start all services when no arguments are provided
if [ $# -eq 0 ]; then
    START_OLLAMA=true
    START_DOCKER=true
fi

while [ "$1" != "" ]; do
    case $1 in
        -h | --help )       show_help
                            ;;
        -o | --ollama )     START_OLLAMA=true
                            ;;
        -d | --docker )     START_DOCKER=true
                            ;;
        -a | --all )        START_OLLAMA=true
                            START_DOCKER=true
                            ;;
        -s | --stop )       STOP_SERVICES=true
                            ;;
        -c | --check )      CHECK_ENVIRONMENT=true
                            ;;
        -l | --list )       LIST_CONTAINERS=true
                            ;;
        -p | --pull )       PULL_IMAGES=true
                            ;;
        --no-pull )         NO_PULL=true
                            START_OLLAMA=true
                            START_DOCKER=true
                            ;;
        -r | --restart )    RESTART_CONTAINER=true
                            CONTAINER_NAME="$2"
                            shift
                            ;;
        -v | --version )    show_version
                            ;;
        * )                 log_error "Unknown option: $1"
                            show_help
                            ;;
    esac
    shift
done

# Perform environment check
if [ "$CHECK_ENVIRONMENT" = true ]; then
    check_environment
    exit $?
fi

# List all containers
if [ "$LIST_CONTAINERS" = true ]; then
    list_containers
    exit $?
fi

# Pull the latest images
if [ "$PULL_IMAGES" = true ]; then
    pull_images
    exit $?
fi

# Restart a specific container
if [ "$RESTART_CONTAINER" = true ]; then
    restart_container "$CONTAINER_NAME"
    exit $?
fi

# Execute service operations
if [ "$STOP_SERVICES" = true ]; then
    # Stop services
    stop_ollama
    OLLAMA_RESULT=$?
    
    stop_docker
    DOCKER_RESULT=$?
    
    # Show summary
    echo ""
    log_info "=== Stop results ==="
    if [ $OLLAMA_RESULT -eq 0 ]; then
        log_success "✓ Ollama service stopped"
    else
        log_error "✗ Failed to stop Ollama service"
    fi
    
    if [ $DOCKER_RESULT -eq 0 ]; then
        log_success "✓ Docker containers stopped"
    else
        log_error "✗ Failed to stop Docker containers"
    fi
    
    log_success "Service shutdown complete."
else
    # Start services
    OLLAMA_RESULT=1
    DOCKER_RESULT=1
    if [ "$START_OLLAMA" = true ]; then
        start_ollama
        OLLAMA_RESULT=$?
    fi
    
    if [ "$START_DOCKER" = true ]; then
        start_docker
        DOCKER_RESULT=$?
    fi
    
    # Show summary
    echo ""
    log_info "=== Start results ==="
    if [ "$START_OLLAMA" = true ]; then
        if [ $OLLAMA_RESULT -eq 0 ]; then
            log_success "✓ Ollama service started"
        else
            log_error "✗ Ollama service failed to start"
        fi
    fi
    
    if [ "$START_DOCKER" = true ]; then
        if [ $DOCKER_RESULT -eq 0 ]; then
            log_success "✓ Docker containers started"
        else
            log_error "✗ Failed to start Docker containers"
        fi
    fi
    
    if [ "$START_OLLAMA" = true ] && [ "$START_DOCKER" = true ]; then
        if [ $OLLAMA_RESULT -eq 0 ] && [ $DOCKER_RESULT -eq 0 ]; then
            log_success "All services are running; access them at:"
            printf "%b\n" "${GREEN}  - Frontend UI: http://localhost:${FRONTEND_PORT:-80}${NC}"
            printf "%b\n" "${GREEN}  - API endpoint: http://localhost:${APP_PORT:-8080}${NC}"
            printf "%b\n" "${GREEN}  - Jaeger tracing: http://localhost:16686${NC}"
            echo ""
            log_info "Streaming container logs (Ctrl+C to stop logging without stopping containers)..."
            "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD logs app docreader postgres --since=10s -f
        else
            log_error "Some services failed to start; review logs to resolve issues."
        fi
    elif [ "$START_OLLAMA" = true ] && [ $OLLAMA_RESULT -eq 0 ]; then
        log_success "Ollama service is running; access at:"
        printf "%b\n" "${GREEN}  - Ollama API: http://localhost:$OLLAMA_PORT${NC}"
    elif [ "$START_DOCKER" = true ] && [ $DOCKER_RESULT -eq 0 ]; then
        log_success "Docker containers are running; access at:"
        printf "%b\n" "${GREEN}  - Frontend UI: http://localhost:${FRONTEND_PORT:-80}${NC}"
        printf "%b\n" "${GREEN}  - API endpoint: http://localhost:${APP_PORT:-8080}${NC}"
        printf "%b\n" "${GREEN}  - Jaeger tracing: http://localhost:16686${NC}"
        echo ""
        log_info "Streaming container logs (Ctrl+C to stop logging without stopping containers)..."
        "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD logs app docreader postgres --since=10s -f
    fi
fi

exit 0
