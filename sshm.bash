#!/usr/bin/env bash

###############################################################################
# Copyright 2024 Guillaume Archambault
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law ou agreed to in writing, software
# distributed under the License est distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OU CONDITIONS OF ANY KIND, either express ou implied.
# See the License for the specific language governing permissions and
# limitations under the License.
###############################################################################

set -eo pipefail; [[ $TRACE ]] && set -x

readonly VERSION="2.1.0"
readonly CONFIG_DIR="${HOME}/.config/sshm"
readonly DEFAULT_CONFIG="${HOME}/.ssh/config"
readonly CURRENT_CONTEXT_FILE="${CONFIG_DIR}/.current_context"
readonly GITHUB_REPO="Gu1llaum-3/sshm"

mkdir -p "$CONFIG_DIR"

# Initialize SSHM_CONTEXT from file or default if not already set
if [[ -z "${SSHM_CONTEXT:-}" ]]; then
    if [[ -f "$CURRENT_CONTEXT_FILE" ]]; then
        export SSHM_CONTEXT=$(cat "$CURRENT_CONTEXT_FILE")
    else
        export SSHM_CONTEXT="$DEFAULT_CONFIG"
    fi
fi

CONFIG_FILE="$SSHM_CONTEXT"

sshm_version() {
  echo "sshm $VERSION"

  # Fetch the latest release tag from GitHub
  local latest_version
  latest_version=$(curl -s "https://api.github.com/repos/$GITHUB_REPO/releases/latest" | jq -r .tag_name)
  
  if [[ "$latest_version" == "null" ]]; then
    echo "sshm $VERSION"
    echo "Error: Unable to fetch the latest release from GitHub." 1>&2
    exit 1
  fi

  # Compare with the current version
  if [[ "$latest_version" != "$VERSION" ]]; then
    echo "A new version of sshm is available: $latest_version (current: $VERSION)"
    echo "You can update by running: git pull origin main"
  else
    echo "This is the latest version"
  fi
}

sshm_help() {
  echo "Usage: sshm [command] <command-specific-options>"
  echo
  echo "Commands:"
  cat<<EOF | column -t -s $'\t'
  <host>                  Connect directly to SSH host by name
  list                    List SSH hosts and prompt for connection
  ping <name>             Ping an SSH host to check availability
  view <name>             Check configuration of host
  delete <name>           Delete an SSH host from the configuration
  add                     Add an SSH host to the configuration
  context list            List available contexts
  context use <name>      Use a specific context
  context create <name>   Create a new context
  context delete <name>   Delete an existing context
  help                    Displays help
  version                 Displays the current version
EOF
}

sshm_list() {
  local config_file="$CONFIG_FILE"
  
  # Check if the file exists and is not empty
  if [[ ! -s "$config_file" ]]; then
    echo -e "\nNo SSH hosts configured in current context."
    echo "Use 'sshm add' to add a new host configuration."
    exit 0
  fi

  # Check if there are any Host entries
  if ! grep -q "^Host " "$config_file"; then
    echo -e "\nNo SSH hosts configured in current context."
    echo "Use 'sshm add' to add a new host configuration."
    exit 0
  fi

  echo -e "\nList of SSH hosts:"
  grep -E '^Host ' "$config_file" | awk '{print $2}' | grep -v '^#' | sort | nl
  
  echo -ne "\nEnter the number or name of the host (or press Enter to exit): "
  read host
  if [[ -z "$host" ]]; then
    echo "No host specified, exiting."
    exit 0
  fi
  
  sshm_connect "$config_file" "$host"
}

sshm_connect() {
  local config_file="$1"
  local host="$2"
  if [[ -z "$host" ]]; then
    echo "Error: please provide a host number or name." 1>&2
    exit 1
  fi

  if [[ "$host" =~ ^[0-9]+$ ]]; then
    local host_name
    host_name=$(grep -E '^Host ' "$config_file" | awk '{print $2}' | grep -v '^#' | sed -n "${host}p")
    if [[ -n "$host_name" ]]; then
      ssh -F "$config_file" "$host_name"
    else
      echo "Error: Invalid host number." 1>&2
      exit 2
    fi
  else
    # Check if the host exists in the SSH configuration
    if ! grep -q "^Host $host$" "$config_file"; then
      echo "Error: Host '$host' not found in SSH configuration." 1>&2
      echo "Use 'sshm list' to see available hosts or 'sshm add $host' to add it." 1>&2
      exit 1
    fi
    
    ssh -F "$config_file" "$host"
  fi
}

sshm_ping() {
  local config_file="$1"
  local host="$2"
  if [[ -z "$host" ]]; then
    echo "Error: please provide a host name." 1>&2
    exit 1
  fi

  local hostname
  hostname=$(awk '/^Host '"$host"'$/,/^$/' "$config_file" | awk '/HostName/ {print $2}')
  if [[ -z "$hostname" ]]; then
    echo "Error: HostName not found for host $host in SSH configuration." 1>&2
    exit 1
  fi

  if ping -c 1 -W 1 "$hostname" &> /dev/null; then
    echo -e "\033[32m$host ($hostname) is available\033[0m"
  else
    echo -e "\033[31m$host ($hostname) is unavailable\033[0m"
  fi
}

sshm_view() {
  local config_file="$1"
  local host="$2"
  if [[ -z "$host" ]]; then
    echo "Error: please provide a host name." 1>&2
    exit 1
  fi

  local host_info
  host_info=$(awk '/^Host '"$host"'$/,/^$/' "$config_file")
  if [[ -z "$host_info" ]]; then
    echo "Error: host not found in SSH configuration." 1>&2
    exit 1
  fi

  echo -e "\nInformation for host $host:\n"
  echo "$host_info"
}

sshm_delete() {
  local config_file="$1"
  local host="$2"
  local silent="${3:-false}"

  if [[ -z "$host" ]]; then
    echo "Error: please provide a host name." 1>&2
    exit 1
  fi

  # Create a backup of the original file
  cp "$config_file" "$config_file.bak"

  # Create a temporary file for the new content
  local tmp_file
  tmp_file=$(mktemp)
  sed '/^Host '"$host"'$/,/^$/d' "$config_file" > "$tmp_file"

  # Check if the temporary file is not empty before overwriting
  if [[ -s "$tmp_file" ]]; then
    mv "$tmp_file" "$config_file"
    rm -f "$config_file.bak"
  else
    mv "$config_file.bak" "$config_file"
    rm -f "$tmp_file"
    echo "Error: Operation would result in empty file. Operation cancelled." 1>&2
    exit 1
  fi

  if [[ "$silent" != "true" ]]; then
    echo "Host $host removed from SSH configuration."
  fi
}

sshm_add() {
  local config_file="$CONFIG_FILE"
  local host="$1"
  local hostname
  local user
  local port
  local identity_file
  local proxy_jump

  default_identity_file=$(find ~/.ssh -maxdepth 1 -type f \( -name "id_rsa" -o -name "id_ed25519" -o -name "id_ecdsa" -o -name "id_dsa" \) | head -n 1)
  default_identity_file=${default_identity_file:-~/.ssh/id_rsa}

  if [[ -z "$host" ]]; then
    read -p "Enter host name: " host
    if [[ -z "$host" ]]; then
      echo "Error: host name cannot be empty." 1>&2
      exit 1
    fi
  fi

  # Vérifier si le host existe déjà
  if grep -q "^Host $host$" "$config_file" 2>/dev/null; then
    echo "Error: Host '$host' already exists in configuration." 1>&2
    echo "Use 'sshm edit $host' to modify the existing configuration or choose a different name." 1>&2
    exit 1
  fi

  read -p "Enter HostName (IP address or domain): " hostname
  if [[ -z "$hostname" ]]; then
    echo "Error: HostName cannot be empty." 1>&2
    exit 1
  fi

  read -p "Enter user name (default: $(whoami)): " user
  user=${user:-$(whoami)}

  read -p "Enter SSH port (default: 22): " port
  port=${port:-22}

  read -p "Enter path to SSH key (default: $default_identity_file): " identity_file
  identity_file=${identity_file:-$default_identity_file}

  read -p "Enter ProxyJump host (optional): " proxy_jump

  # Create the file if it doesn't exist
  touch "$config_file"

  {
    echo ""
    echo "Host $host"
    echo "    HostName $hostname"
    echo "    User $user"
    if [[ "$port" -ne 22 ]]; then
      echo "    Port $port"
    fi
    if [[ "$identity_file" != "$default_identity_file" ]]; then
      echo "    IdentityFile $identity_file"
    fi
    if [[ -n "$proxy_jump" ]]; then
      echo "    ProxyJump $proxy_jump"
    fi
  } >> "$config_file"

  echo "Configuration for host $host added successfully."
}

sshm_edit() {
  local config_file="$CONFIG_FILE"
  local host="$1"
  if [[ -z "$host" ]]; then
    echo "Error: please provide a host name." 1>&2
    exit 1
  fi

  local host_info
  host_info=$(awk '/^Host '"$host"'$/,/^$/' "$config_file")
  if [[ -z "$host_info" ]]; then
    echo "Error: host not found in SSH configuration." 1>&2
    exit 1
  fi

  default_identity_file=$(find ~/.ssh -maxdepth 1 -type f \( -name "id_rsa" -o -name "id_ed25519" -o -name "id_ecdsa" -o -name "id_dsa" \) | head -n 1)
  default_identity_file=${default_identity_file:-~/.ssh/id_rsa}

  # Extract current values
  local current_hostname=$(echo "$host_info" | awk '/HostName/ {print $2}')
  local current_user=$(echo "$host_info" | awk '/User/ {print $2}')
  local current_port=$(echo "$host_info" | awk '/Port/ {print $2}')
  local current_identity_file=$(echo "$host_info" | awk '/IdentityFile/ {print $2}')
  local current_proxyjump=$(echo "$host_info" | awk '/ProxyJump/ {print $2}')

  # Create backup of the original file
  cp "$config_file" "$config_file.bak"

  # Prompt for new values, defaulting to current values if no input is given
  read -p "HostName [$current_hostname]: " new_hostname
  new_hostname=${new_hostname:-$current_hostname}

  read -p "User [$current_user]: " new_user
  new_user=${new_user:-$current_user}

  read -p "Port [${current_port:-22}]: " new_port
  new_port=${new_port:-${current_port:-22}}

  read -p "IdentityFile [${current_identity_file:-$default_identity_file}]: " new_identity_file
  new_identity_file=${new_identity_file:-${current_identity_file:-$default_identity_file}}

  if [[ -n "$current_proxyjump" ]]; then
    read -p "ProxyJump [$current_proxyjump]: " new_proxyjump
    new_proxyjump=${new_proxyjump:-$current_proxyjump}
  else
    read -p "ProxyJump (leave empty if none): " new_proxyjump
  fi
  
  # Create a temporary file for the new content
  local tmp_file
  tmp_file=$(mktemp)
  
  # Delete the old configuration
  sed '/^Host '"$host"'$/,/^$/d' "$config_file" > "$tmp_file"

  # Check if the temporary file is not empty
  if [[ ! -s "$tmp_file" ]]; then
    mv "$config_file.bak" "$config_file"
    rm -f "$tmp_file"
    echo "Error: Operation would result in empty file. Operation cancelled." 1>&2
    exit 1
  fi

  # Add the new configuration
  {
    echo ""
    echo "Host $host"
    echo "    HostName $new_hostname"
    echo "    User $new_user"
    if [[ "$new_port" -ne 22 ]]; then
      echo "    Port $new_port"
    fi
    if [[ "$new_identity_file" != "$default_identity_file" ]]; then
      echo "    IdentityFile $new_identity_file"
    fi
    if [[ -n "$new_proxyjump" ]]; then
      echo "    ProxyJump $new_proxyjump"
    fi
  } >> "$tmp_file"

  # Move the temporary file to the final location
  mv "$tmp_file" "$config_file"
  rm -f "$config_file.bak"

  echo "Configuration for host $host updated successfully."
}

context_list() {
  echo "Available contexts:"
  if [[ "$SSHM_CONTEXT" == "$DEFAULT_CONFIG" ]]; then
    echo "* default"
  else
    echo "  default"
  fi

  for context in "$CONFIG_DIR"/*; do
    if [[ -f "$context" ]]; then
      local context_name
      context_name=$(basename "$context")
      if [[ "$CONFIG_DIR/$context_name" == "$SSHM_CONTEXT" ]]; then
        echo "* $context_name"
      else
        echo "  $context_name"
      fi
    fi
  done
}

context_use() {
  local context="$1"
  if [[ -z "$context" ]]; then
    echo "Error: please provide a context name." 1>&2
    exit 1
  fi

  if [[ "$context" == "default" ]]; then
    export SSHM_CONTEXT="$DEFAULT_CONFIG"
  elif [[ ! -f "$CONFIG_DIR/$context" ]]; then
    echo "Error: context '$context' does not exist." 1>&2
    exit 1
  else
    export SSHM_CONTEXT="$CONFIG_DIR/$context"
  fi

  # Update the file for persistence between sessions
  echo "$SSHM_CONTEXT" > "$CURRENT_CONTEXT_FILE"
  echo "Switched to context '$context'."
  
  # Update CONFIG_FILE for the current session
  CONFIG_FILE="$SSHM_CONTEXT"
}

context_create() {
  local context="$1"
  if [[ -z "$context" ]]; then
    echo "Error: please provide a context name." 1>&2
    exit 1
  fi

  if [[ -f "$CONFIG_DIR/$context" ]]; then
    echo "Error: context '$context' already exists." 1>&2
    exit 1
  fi

  touch "$CONFIG_DIR/$context"
  chmod 600 "$CONFIG_DIR/$context"
  echo "Context '$context' created."
}

context_delete() {
  local context="$1"
  if [[ -z "$context" ]]; then
    echo "Error: please provide a context name." 1>&2
    exit 1
  fi

  if [[ ! -f "$CONFIG_DIR/$context" ]]; then
    echo "Error: context '$context' does not exist." 1>&2
    exit 1
  fi

  rm -f "$CONFIG_DIR/$context"
  echo "Context '$context' deleted."

  # If the deleted context was the current one, switch to default
  if [[ "$SSHM_CONTEXT" == "$CONFIG_DIR/$context" ]]; then
    export SSHM_CONTEXT="$DEFAULT_CONFIG"
    echo "$SSHM_CONTEXT" > "$CURRENT_CONTEXT_FILE"
    CONFIG_FILE="$SSHM_CONTEXT"
    echo "Switched to default context."
  fi
}

sshm_main() {
  local command="$1"
  shift

  if [[ -z $command ]]; then
    sshm_version
    echo
    sshm_help
    exit 0
  fi

  # Check if command is a known command, otherwise treat it as a host to connect to
  case "$command" in
    "list")
      sshm_list
      ;;
    "ping")
      sshm_ping "$CONFIG_FILE" "$@"
      ;;
    "view")
      sshm_view "$CONFIG_FILE" "$@"
      ;;
    "delete")
      sshm_delete "$CONFIG_FILE" "$@"
      ;;
    "add")
      sshm_add "$@"
      ;;
    "edit")
      sshm_edit "$@"
      ;;
    "context")
      local subcommand="$1"
      shift
      case "$subcommand" in
        "list")
          context_list "$@"
          ;;
        "use")
          context_use "$@"
          ;;
        "create")
          context_create "$@"
          ;;
        "delete")
          context_delete "$@"
          ;;
        *)
          echo "Error: invalid context subcommand." 1>&2
          sshm_help
          exit 3
          ;;
      esac
      ;;
    "version")
      sshm_version
      ;;
    "help")
      sshm_help
      ;;
    *)
      # If command is not recognized, treat it as a host name to connect to
      sshm_connect "$CONFIG_FILE" "$command"
  esac
}

if [[ "$0" == "$BASH_SOURCE" ]]; then
  # If no arguments are provided, display help
  if [[ $# -eq 0 ]]; then
    sshm_version
    echo
    sshm_help
  else
    sshm_main "$@"
  fi
fi
