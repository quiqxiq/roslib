# Commands Mapping

Peta dari analisis `mikhmonv3-analisis.md` ke fungsi Go.

## §1.1 System Info & Control

| Command | Fungsi |
|---|---|
| `/system/identity/print` | `system.Client.Identity` |
| `/system/resource/print` | `system.Client.Resource` |
| `/system/routerboard/print` | `system.Client.Routerboard` |
| `/system/clock/print` | `system.Client.Clock` |
| `/system/reboot` | `system.Client.Reboot` |
| `/system/shutdown` | `system.Client.Shutdown` |

## §1.2 System Logging

| Command | Fungsi |
|---|---|
| `/system/logging/print ?prefix=…` | `system.Client.LoggingByPrefix` |
| `/system/logging/add` (hotspot disk) | `system.Client.LoggingAddHotspotDisk` |

## §1.3 System Script

| Command | Fungsi |
|---|---|
| `/system/script/print` | `system.Client.ScriptList` |
| `/system/script/print ?name=…` | `system.Client.ScriptByName` |
| `/system/script/print ?comment=…` | `system.Client.ScriptByComment` |
| `/system/script/print ?owner=…` | `system.Client.ScriptByOwner` |
| `/system/script/print ?source=…` | `system.Client.ScriptBySource` |
| `/system/script/print ?.id=…` | `system.Client.ScriptByID` |
| `/system/script/add` | `system.Client.ScriptAdd` |
| `/system/script/set` | `system.Client.ScriptSet` |
| `/system/script/remove` | `system.Client.ScriptRemove` |

Konstanta filter:
- `system.CommentTransaction = "mikhmon"` (for transaction reports)
- `system.CommentQuickPrint = "QuickPrintMikhmon"` (for quick print configs)

## §1.4 System Scheduler

| Command | Fungsi |
|---|---|
| `/system/scheduler/print` | `system.Client.SchedulerList` |
| `/system/scheduler/print count-only=""` | `system.Client.SchedulerCount` |
| `/system/scheduler/print ?name=…` | `system.Client.SchedulerByName` |
| `/system/scheduler/add` | `system.Client.SchedulerAdd` |
| `/system/scheduler/set` | `system.Client.SchedulerSet` |
| `/system/scheduler/remove` | `system.Client.SchedulerRemove` |

## §1.5 Log

| Command | Fungsi |
|---|---|
| `/log/print` | `syslog.Client.LogList` |
| `/log/print ?topics=…` | `syslog.Client.LogByTopics` |

## §1.6 IP Hotspot User

| Command | Fungsi |
|---|---|
| `/ip/hotspot/user/print` | `hotspot.Client.UserList` |
| `/ip/hotspot/user/print count-only=""` | `hotspot.Client.UserCount` |
| `/ip/hotspot/user/print ?name=…` | `hotspot.Client.UserByName` |
| `/ip/hotspot/user/print ?.id=…` | `hotspot.Client.UserByID` |
| `/ip/hotspot/user/print ?profile=…` | `hotspot.Client.UserByProfile` |
| `/ip/hotspot/user/print ?comment=…` | `hotspot.Client.UserByComment` |
| `/ip/hotspot/user/add` | `hotspot.Client.UserAdd` |
| `/ip/hotspot/user/set` (full) | `hotspot.Client.UserSet` |
| `/ip/hotspot/user/set =disabled=…` | `hotspot.Client.UserSetDisabled` |
| `/ip/hotspot/user/set =comment=<exp>` | `hotspot.Client.UserSetExpiry` |
| `/ip/hotspot/user/set =mac-address=…` | `hotspot.Client.UserSetMAC` |
| `/ip/hotspot/user/set =limit-uptime=0 =comment=` | `hotspot.Client.UserResetUsage` |
| `/ip/hotspot/user/remove` | `hotspot.Client.UserRemove` |
| `/ip/hotspot/user/reset-counters` | `hotspot.Client.UserResetCounters` |

## §1.7 IP Hotspot User Profile

| Command | Fungsi |
|---|---|
| `/ip/hotspot/user/profile/print` | `hotspot.Client.ProfileList` |
| `/ip/hotspot/user/profile/print ?name=…` | `hotspot.Client.ProfileByName` |
| `/ip/hotspot/user/profile/print ?.id=…` | `hotspot.Client.ProfileByID` |
| `/ip/hotspot/user/profile/add` | `hotspot.Client.ProfileAdd` |
| `/ip/hotspot/user/profile/set` | `hotspot.Client.ProfileSet` |
| `/ip/hotspot/user/profile/remove` | `hotspot.Client.ProfileRemove` |

## §1.8 Active / Host / Cookie / Server

| Command | Fungsi |
|---|---|
| `/ip/hotspot/print` | `hotspot.Client.ServerList` |
| `/ip/hotspot/active/print` | `hotspot.Client.ActiveList` |
| `/ip/hotspot/active/print count-only=""` | `hotspot.Client.ActiveCount` |
| `/ip/hotspot/active/print ?server=…` | `hotspot.Client.ActiveByServer` |
| `/ip/hotspot/active/print count-only="" ?server=…` | `hotspot.Client.ActiveCountByServer` |
| `/ip/hotspot/active/print ?.id=…` | `hotspot.Client.ActiveByID` |
| `/ip/hotspot/active/remove` | `hotspot.Client.ActiveRemove` |
| `/ip/hotspot/host/print` | `hotspot.Client.HostList` |
| `/ip/hotspot/host/remove` | `hotspot.Client.HostRemove` |
| `/ip/hotspot/cookie/print` | `hotspot.Client.CookieList` |
| `/ip/hotspot/cookie/print count-only=""` | `hotspot.Client.CookieCount` |
| `/ip/hotspot/cookie/print ?user=…` | `hotspot.Client.CookieByUser` |
| `/ip/hotspot/cookie/remove` | `hotspot.Client.CookieRemove` |

## §1.9 IP Binding

| Command | Fungsi |
|---|---|
| `/ip/hotspot/ip-binding/print` | `hotspot.Client.BindingList` |
| `/ip/hotspot/ip-binding/print ?.id=…` | `hotspot.Client.BindingByID` |
| `/ip/hotspot/ip-binding/print count-only=""` | `hotspot.Client.BindingCount` |
| `/ip/hotspot/ip-binding/set =type=…` | `hotspot.Client.BindingSetType` |
| `/ip/hotspot/ip-binding/set =disabled=…` | `hotspot.Client.BindingSetDisabled` |
| `/ip/hotspot/ip-binding/remove` | `hotspot.Client.BindingRemove` |

## §1.10 Queue / Pool / ARP / DHCP

| Command | Fungsi |
|---|---|
| `/queue/simple/print` | `network.Client.QueueSimpleList` |
| `/queue/simple/print ?dynamic=false` | `network.Client.QueueSimpleStatic` |
| `/queue/simple/print ?name=…` | `network.Client.QueueSimpleByName` |
| `/queue/simple/remove` | `network.Client.QueueSimpleRemove` |
| `/ip/pool/print` | `network.Client.IPPoolList` |
| `/ip/arp/print ?mac-address=…` | `network.Client.ARPByMAC` |
| `/ip/arp/remove` | `network.Client.ARPRemove` |
| `/ip/dhcp-server/lease/print` | `network.Client.DHCPLeaseList` |
| `/ip/dhcp-server/lease/print count-only=""` | `network.Client.DHCPLeaseCount` |
| `/ip/dhcp-server/lease/print ?mac-address=…` | `network.Client.DHCPLeaseByMAC` |
| `/ip/dhcp-server/lease/remove` | `network.Client.DHCPLeaseRemove` |

## §1.11 Interface & Traffic

| Command | Fungsi |
|---|---|
| `/interface/print` | `network.Client.InterfaceList` |
| `/interface/monitor-traffic once=""` | `network.Client.MonitorTrafficSnapshot` |

## §1.12 PPP (inferred)

| Command | Fungsi |
|---|---|
| `/ppp/secret/print` | `ppp.Client.SecretList` |
| `/ppp/secret/print ?name=…` | `ppp.Client.SecretByName` |
| `/ppp/secret/add` | `ppp.Client.SecretAdd` |
| `/ppp/secret/set` | `ppp.Client.SecretSet` |
| `/ppp/secret/set =disabled=…` | `ppp.Client.SecretSetDisabled` |
| `/ppp/secret/remove` | `ppp.Client.SecretRemove` |
| `/ppp/profile/print` | `ppp.Client.ProfileList` |
| `/ppp/profile/print ?name=…` | `ppp.Client.ProfileByName` |
| `/ppp/profile/add` | `ppp.Client.ProfileAdd` |
| `/ppp/profile/set` | `ppp.Client.ProfileSet` |
| `/ppp/profile/remove` | `ppp.Client.ProfileRemove` |
| `/ppp/active/print` | `ppp.Client.ActiveList` |
| `/ppp/active/remove` | `ppp.Client.ActiveRemove` (file PHP-nya ADA) |
