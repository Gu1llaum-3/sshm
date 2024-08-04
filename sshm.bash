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
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
###############################################################################

set -eo pipefail; [[ $TRACE ]] && set -x

readonly VERSION="1.0.0"
CONFIG_FILE=${SSHM_CONFIG:-~/.ssh/config}

ssh_manager_version() {
  echo "ssh_manager $VERSION"
}

ssh_manager_help() {
  echo "Usage: ssh_manager command [--config <file>|-c <file>] <command-specific-options>"
  echo
  echo "Commands:"
  cat<<EOF | column -t -s $'\t'
  list                    List SSH hosts and prompt for connection
  connect <number|name>   Connect to SSH host by number or name
  ping <name>             Ping an SSH host to check availability
  view <name>             Check configuration of host
  delete <name>           Delete an SSH host from the configuration
  add                     Add an SSH host to the configuration
  help                    Displays help
  version                 Displays the current version
EOF
  echo
  echo "Flags:"
  cat<<EOF | column -t -s $'\t'
  -c, --config            Select a specific ssh config file
EOF
  echo
  echo "Environment Variables:"
  cat<<EOF | column -t -s $'\t'
  SSHM_CONFIG             Specify the path of an ssh config file"
EOF
}

ssh_manager_list() {
  local config_file="$1"
  echo -e "\nList of SSH hosts:"
  grep -E '^Host ' "$config_file" | awk '{print $2}' | grep -v '^#' | sort | nl
  
  echo -ne "\nEnter the number or name of the host (or press Enter to exit): "
  read host
  if [[ -z "$host" ]]; then
    echo "No host specified, exiting."
    exit 0
  fi
  
  ssh_manager_connect "$config_file" "$host"
}

ssh_manager_connect() {
  local config_file="$1"
  local host="$2"
  if [[ -z "$host" ]]; then
    echo "Error: please provide a host number or name." 1>&2
    exit 1
  fi
  if [[ "$host" =~ ^[0-9]+$ ]]; then
    local host_name
    host_name=$(grep -E '^Host ' "$config_file" | awk '{print $2}' | grep -v '^#' | sed -n "${host}p")
    if [ -n "$host_name" ]; then
      ssh "$host_name"
    else
      echo "Invalid number." 1>&2
      exit 2
    fi
  else
    ssh "$host"
  fi
}

ssh_manager_ping() {
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

ssh_manager_view() {
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

ssh_manager_delete() {
  local config_file="$1"
  local host="$2"
  if [[ -z "$host" ]]; then
    echo "Error: please provide a host name." 1>&2
    exit 1
  fi

  local tmp_file=$(mktemp)
  sed '/^Host '"$host"'$/,/^$/d' "$config_file" > "$tmp_file"
  mv "$tmp_file" "$config_file"
  echo "Host $host removed from SSH configuration."
}

ssh_manager_add() {
  local config_file="$1"
  local host="$2"
  local hostname
  local user
  local port
  local identity_file

  # Request necessary information
  if [[ -z "$host" ]]; then
    read -p "Enter host name: " host
    if [[ -z "$host" ]]; then
      echo "Error: host name cannot be empty." 1>&2
      exit 1
    fi
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

  read -p "Enter path to SSH key (default: ~/.ssh/id_rsa): " identity_file
  identity_file=${identity_file:-~/.ssh/id_rsa}

  # Add the new configuration to the file
  {
    echo ""
    echo "Host $host"
    echo "    HostName $hostname"
    echo "    User $user"
    if [[ "$port" -ne 22 ]]; then
      echo "    Port $port"
    fi
    if [[ "$identity_file" != ~/.ssh/id_rsa ]]; then
      echo "    IdentityFile $identity_file"
    fi
  } >> "$config_file"

  echo "Configuration for host $host added successfully."
}

ssh_manager_edit() {
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

  # Extract current values
  local current_hostname=$(echo "$host_info" | awk '/HostName/ {print $2}')
  local current_user=$(echo "$host_info" | awk '/User/ {print $2}')
  local current_port=$(echo "$host_info" | awk '/Port/ {print $2}')
  local current_identity_file=$(echo "$host_info" | awk '/IdentityFile/ {print $2}')

  # Prompt for new values, defaulting to current values if no input is given
  read -p "HostName [$current_hostname]: " new_hostname
  new_hostname=${new_hostname:-$current_hostname}

  read -p "User [$current_user]: " new_user
  new_user=${new_user:-$current_user}

  read -p "Port [${current_port:-22}]: " new_port
  new_port=${new_port:-${current_port:-22}}

  read -p "IdentityFile [${current_identity_file:-~/.ssh/id_rsa}]: " new_identity_file
  new_identity_file=${new_identity_file:-${current_identity_file:-~/.ssh/id_rsa}}

  # Delete the old configuration
  ssh_manager_delete "$config_file" "$host"

  # Add the new configuration
  {
    echo ""
    echo "Host $host"
    echo "    HostName $new_hostname"
    echo "    User $new_user"
    if [[ "$new_port" -ne 22 ]]; then
      echo "    Port $new_port"
    fi
    if [[ "$new_identity_file" != ~/.ssh/id_rsa ]]; then
      echo "    IdentityFile $new_identity_file"
    fi
  } >> "$config_file"

  echo "Configuration for host $host updated successfully."
}

ssh_manager_main() {
  local config_file="$CONFIG_FILE"
  local command="$1"
  shift

  while [[ "$#" -gt 0 ]]; do
    case "$1" in
      --config|-c)
        config_file="$2"
        shift 2
        ;;
      *)
        break
        ;;
    esac
  done

  if [[ -z $command ]]; then
    ssh_manager_version
    echo
    ssh_manager_help
    exit 0
  fi

  case "$command" in
    "list")
      ssh_manager_list "$config_file"
      ;;
    "connect")
      ssh_manager_connect "$config_file" "$@"
      ;;
    "ping")
      ssh_manager_ping "$config_file" "$@"
      ;;
    "view")
      ssh_manager_view "$config_file" "$@"
      ;;
    "delete")
      ssh_manager_delete "$config_file" "$@"
      ;;
    "add")
      ssh_manager_add "$config_file" "$@"
      ;;
    "edit")
      ssh_manager_edit "$config_file" "$@"
      ;;
    "version")
      ssh_manager_version
      ;;
    "help")
      ssh_manager_help
      ;;
    *)
      ssh_manager_help 1>&2
      exit 3
  esac
}

if [[ "$0" == "$BASH_SOURCE" ]]; then
  ssh_manager_main "$@"
fi
