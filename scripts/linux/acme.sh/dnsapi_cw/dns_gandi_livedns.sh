#!/usr/bin/env sh

ABS_CURR_PATH=$(dirname $(realpath "${BASH_SOURCE[0]}"))
SRC_FILE="${ABS_CURR_PATH}/../acme_src.sh"
. "${SRC_FILE}"

# shellcheck disable=SC2034
dns_gandi_livedns_info='Gandi.net LiveDNS
Site: Gandi.net/domain/dns
Docs: github.com/acmesh-official/acme.sh/wiki/dnsapi#dns_gandi_livedns
Options:
 GANDI_LIVEDNS_KEY API Key
Issues: github.com/fcrozat/acme.sh
Author: Frédéric Crozat <fcrozat@suse.com>, Dominik Röttsches <drott@google.com>
'

# Gandi LiveDNS v5 API
# https://api.gandi.net/docs/livedns/
# https://api.gandi.net/docs/authentication/ for token + apikey (deprecated) authentication
# currently under beta

########  Public functions #####################

GANDI_LIVEDNS_API="https://api.gandi.net/v5/livedns"

#Usage: dns_gandi_livedns_add   _acme-challenge.www.domain.com   "XKrxpRBosdIKFzxW_CT3KLZNf6q0HG9i01zxXp5CPBs"
dns_gandi_livedns_add() {
  fulldomain=$1
  txtvalue=$2

  if [ -z "$GANDI_LIVEDNS_KEY" ] && [ -z "$GANDI_LIVEDNS_TOKEN" ]; then
    _err "No Token or API key (deprecated) specified for Gandi LiveDNS."
    _err "Create your token or key and export it as GANDI_LIVEDNS_KEY or GANDI_LIVEDNS_TOKEN respectively"
    return 1
  fi

  # Keep only one secret in configuration
  if [ -n "$GANDI_LIVEDNS_TOKEN" ]; then
    _saveaccountconf GANDI_LIVEDNS_TOKEN "$GANDI_LIVEDNS_TOKEN"
    _clearaccountconf GANDI_LIVEDNS_KEY
  elif [ -n "$GANDI_LIVEDNS_KEY" ]; then
    _saveaccountconf GANDI_LIVEDNS_KEY "$GANDI_LIVEDNS_KEY"
    _clearaccountconf GANDI_LIVEDNS_TOKEN
  fi

  _debug "First detect the root zone"
  if ! _get_root "$fulldomain"; then
    _err "invalid domain"
    return 1
  fi
  _debug fulldomain "$fulldomain"
  _debug txtvalue "$txtvalue"
  _debug domain "$_domain"
  _debug sub_domain "$_sub_domain"

  _dns_gandi_append_record "$_domain" "$_sub_domain" "$txtvalue"
}

#Usage: fulldomain txtvalue
#Remove the txt record after validation.
dns_gandi_livedns_rm() {
  fulldomain=$1
  txtvalue=$2

  _debug "First detect the root zone"
  if ! _get_root "$fulldomain"; then
    _err "invalid domain"
    return 1
  fi

  _debug fulldomain "$fulldomain"
  _debug domain "$_domain"
  _debug sub_domain "$_sub_domain"
  _debug txtvalue "$txtvalue"

  if ! _dns_gandi_existing_rrset_values "$_domain" "$_sub_domain"; then
    return 1
  fi
  _new_rrset_values=$(echo "$_rrset_values" | sed "s/...$txtvalue...//g")
  # Cleanup dangling commata.
  _new_rrset_values=$(echo "$_new_rrset_values" | sed "s/, ,/ ,/g")
  _new_rrset_values=$(echo "$_new_rrset_values" | sed "s/, *\]/\]/g")
  _new_rrset_values=$(echo "$_new_rrset_values" | sed "s/\[ *,/\[/g")
  _debug "New rrset_values" "$_new_rrset_values"

  _gandi_livedns_rest PUT \
    "domains/$_domain/records/$_sub_domain/TXT" \
    "{\"rrset_ttl\": 300, \"rrset_values\": $_new_rrset_values}" &&
    _contains "$response" '{"message":"DNS Record Created"}' &&
    _info "Removing record $(__green "success")"
}

####################  Private functions below ##################################
#_acme-challenge.www.domain.com
#returns
# _sub_domain=_acme-challenge.www
# _domain=domain.com
_get_root() {
  domain=$1
  i=2
  p=1
  while true; do
    h=$(printf "%s" "$domain" | cut -d . -f "$i"-100)
    _debug h "$h"
    if [ -z "$h" ]; then
      #not valid
      return 1
    fi

    if ! _gandi_livedns_rest GET "domains/$h"; then
      return 1
    fi

    if _contains "$response" '"code": 401'; then
      _err "$response"
      return 1
    elif _contains "$response" '"code": 404'; then
      _debug "$h not found"
    else
      _sub_domain=$(printf "%s" "$domain" | cut -d . -f 1-"$p")
      _domain="$h"
      return 0
    fi
    p="$i"
    i=$(_math "$i" + 1)
  done
  return 1
}

_dns_gandi_append_record() {
  domain=$1
  sub_domain=$2
  txtvalue=$3

  if _dns_gandi_existing_rrset_values "$domain" "$sub_domain"; then
    _debug "Appending new value"
    _rrset_values=$(echo "$_rrset_values" | sed "s/\"]/\",\"$txtvalue\"]/")
  else
    _debug "Creating new record" "$_rrset_values"
    _rrset_values="[\"$txtvalue\"]"
  fi
  _debug new_rrset_values "$_rrset_values"
  _gandi_livedns_rest PUT "domains/$_domain/records/$sub_domain/TXT" \
    "{\"rrset_ttl\": 300, \"rrset_values\": $_rrset_values}" &&
    _contains "$response" '{"message":"DNS Record Created"}' &&
    _info "Adding record $(__green "success")"
}

_dns_gandi_existing_rrset_values() {
  domain=$1
  sub_domain=$2
  if ! _gandi_livedns_rest GET "domains/$domain/records/$sub_domain"; then
    return 1
  fi
  if ! _contains "$response" '"rrset_type":"TXT"'; then
    _debug "Does not have a _acme-challenge TXT record yet."
    return 1
  fi
  if _contains "$response" '"rrset_values":\[\]'; then
    _debug "Empty rrset_values for TXT record, no previous TXT record."
    return 1
  fi
  _debug "Already has TXT record."
  _rrset_values=$(echo "$response" | _egrep_o 'rrset_values.*\[.*\]' |
    _egrep_o '\[".*\"]')
  return 0
}

_gandi_livedns_rest() {
  m=$1
  ep="$2"
  data="$3"
  _debug "$ep"

  export _H1="Content-Type: application/json"

  if [ -n "$GANDI_LIVEDNS_TOKEN" ]; then
    export _H2="Authorization: Bearer $GANDI_LIVEDNS_TOKEN"
  else
    export _H2="Authorization: Apikey $GANDI_LIVEDNS_KEY"
  fi

  if [ "$m" = "GET" ]; then
    response="$(_get "$GANDI_LIVEDNS_API/$ep")"
  else
    _debug data "$data"
    response="$(_post "$data" "$GANDI_LIVEDNS_API/$ep" "" "$m")"
  fi

  if [ "$?" != "0" ]; then
    _err "error $ep"
    return 1
  fi
  _debug2 response "$response"
  return 0
}
