#!/usr/bin/env bash
# ============================================================
#  LibreQuenda installer
#
#    install.sh         sets up (or reconfigures) a server: asks what the .env needs,
#                       generates the secrets, starts the stack, and offers your own
#                       Android app for it
#    install.sh app     only builds your own Android app (again, e.g. for an update)
#
#  Needs bash, whiptail (in Debian and Ubuntu already; newt in Fedora), Docker with
#  Compose v2 and curl. Run it in the directory that holds (or will hold) the
#  installation, or set LIBREQUENDA_DIR.
#  Without a checkout: curl -fsSL https://raw.githubusercontent.com/TriboSolutions/LibreQuenda/main/install.sh | bash
#
#  LIBREQUENDA_LANG    the installer's language: gl, es, en, ca or pt (asked when unset)
#  LIBREQUENDA_REF     the branch or tag the files and the app come from (main)
#  LIBREQUENDA_SOURCE  a local checkout to use instead of downloading it
# ============================================================

# started with sh: dash has no arrays nor ERR traps, so this runs again in bash
if [ -z "${BASH_VERSION:-}" ]; then
	case $0 in *.sh) [ -f "$0" ] && exec bash "$0" "$@" ;; esac
	echo "LibreQuenda's installer runs in bash: bash install.sh" >&2
	exit 1
fi

set -euo pipefail
LOCALE_LANG=${LC_ALL:-${LC_MESSAGES:-${LANG:-}}} # for the installer's language, before the line below changes it
# whiptail draws accents only in a UTF-8 locale (cloud images often start in C, or in one not generated)
if [ "$(locale charmap 2>/dev/null)" != UTF-8 ]; then
	for l in C.UTF-8 C.utf8 en_US.UTF-8 en_US.utf8; do
		if locale -a 2>/dev/null | grep -qx "$l"; then export LC_ALL=$l; break; fi
	done
fi
trap 'printf "\e[31mx %s %s: %s\e[0m\n" "${M[stopped]:-stopped at line}" "$LINENO" "$BASH_COMMAND" >&2' ERR

REPO=TriboSolutions/LibreQuenda
REF=${LIBREQUENDA_REF:-main}
SOURCE=${LIBREQUENDA_SOURCE:-}

bold=$'\e[1m' dim=$'\e[2m' red=$'\e[31m' green=$'\e[32m' yellow=$'\e[33m' off=$'\e[0m'
[ -t 1 ] || { bold='' dim='' red='' green='' yellow='' off=''; }

say() { printf '%s\n' "$*"; }
title() { printf '\n%s== %s ==%s\n' "$bold" "$*" "$off"; }
note() { printf '%s%s%s\n' "$dim" "$*" "$off"; }
warn() { printf '%s! %s%s\n' "$yellow" "$*" "$off"; }
die() { printf '%sx %s%s\n' "$red" "$*" "$off" >&2; exit 1; }
ok() { printf '%s✓ %s%s\n' "$green" "$*" "$off"; }

# ============================================================
#  Languages: m key gl es en ca pt
# ============================================================
declare -A M=()
LI=2
t() { local f=${M[$1]:-$1}; shift; printf -- "$f" "$@"; } # t key [printf arguments]
m() { local k=$1; shift; local v=("$@"); M[$k]=${v[$LI]}; }

messages() {
	m stopped "detido na liña" "detenido en la línea" "stopped at line" "aturat a la línia" "parado na linha"
	m backtitle "LibreQuenda · instalador" "LibreQuenda · instalador" "LibreQuenda · installer" "LibreQuenda · instal·lador" "LibreQuenda · instalador"
	m ok "Aceptar" "Aceptar" "OK" "D'acord" "OK"
	m cancel "Saír" "Salir" "Quit" "Surt" "Sair"
	m yes "Si" "Sí" "Yes" "Sí" "Sim"
	m no "Non" "No" "No" "No" "Não"
	m quit_q "Saír do instalador? Aínda non se cambiou nada." \
		"¿Salir del instalador? Todavía no se ha cambiado nada." \
		"Leave the installer? Nothing has been changed yet." \
		"Vols sortir de l'instal·lador? Encara no s'ha canviat res." \
		"Sair do instalador? Ainda não foi alterado nada."
	m quit_done "Saíches sen cambiar nada." "Has salido sin cambiar nada." "Left without changing anything." "Has sortit sense canviar res." "Saiu sem alterar nada."
	m need_cmd "Precísase %s: %s" "Se necesita %s: %s" "%s is needed: %s" "Cal %s: %s" "É necessário %s: %s"
	m need_compose "Precísase Docker Compose v2 (docker compose …)" "Se necesita Docker Compose v2 (docker compose …)" \
		"Docker Compose v2 is needed (docker compose …)" "Cal Docker Compose v2 (docker compose …)" "É necessário o Docker Compose v2 (docker compose …)"
	m need_docker "Docker non está en marcha, ou este usuario non pode usalo (sudo, ou o grupo docker)" \
		"Docker no está en marcha, o este usuario no puede usarlo (sudo, o el grupo docker)" \
		"Docker is not running, or this user cannot use it (sudo, or the docker group)" \
		"Docker no s'està executant, o aquest usuari no el pot fer servir (sudo, o el grup docker)" \
		"O Docker não está em execução, ou este utilizador não o pode usar (sudo, ou o grupo docker)"
	m why_curl "para descargar os ficheiros de LibreQuenda" "para descargar los archivos de LibreQuenda" \
		"to download LibreQuenda's files" "per baixar els fitxers de LibreQuenda" "para transferir os ficheiros do LibreQuenda"
	m dl_fail "Non se puido descargar %s (%s)" "No se ha podido descargar %s (%s)" "Could not download %s (%s)" "No s'ha pogut baixar %s (%s)" "Não foi possível transferir %s (%s)"
	m src_dl "Descargando o código de LibreQuenda (%s)…" "Descargando el código de LibreQuenda (%s)…" "Downloading LibreQuenda's source (%s)…" "Baixant el codi de LibreQuenda (%s)…" "A transferir o código do LibreQuenda (%s)…"
	m unknown_cmd "comando descoñecido: %s (install.sh, ou install.sh app)" "comando desconocido: %s (install.sh, o install.sh app)" \
		"unknown command: %s (install.sh, or install.sh app)" "ordre desconeguda: %s (install.sh, o install.sh app)" "comando desconhecido: %s (install.sh, ou install.sh app)"

	m nav_hint "Frechas e Tab para moverte, Intro para aceptar, Esc para saír. Cada campo trae o valor actual: edítao ou acéptao tal cal." \
		"Flechas y Tab para moverte, Intro para aceptar, Esc para salir. Cada campo trae el valor actual: edítalo o acéptalo tal cual." \
		"Arrows and Tab to move, Enter to accept, Esc to leave. Each field holds its current value: edit it or accept it as it is." \
		"Fletxes i Tab per moure't, Retorn per acceptar, Esc per sortir. Cada camp porta el valor actual: edita'l o accepta'l tal com és." \
		"Setas e Tab para navegar, Enter para aceitar, Esc para sair. Cada campo traz o valor atual: edite-o ou aceite-o como está."
	m new_install "Vaise instalar LibreQuenda en %s." "Se va a instalar LibreQuenda en %s." "LibreQuenda will be installed in %s." "S'instal·larà LibreQuenda a %s." "O LibreQuenda vai ser instalado em %s."
	m found_install "Hai unha instalación en %s: podes cambiar a súa configuración." "Hay una instalación en %s: puedes cambiar su configuración." \
		"There is an installation in %s: you can change its configuration." "Hi ha una instal·lació a %s: en pots canviar la configuració." "Existe uma instalação em %s: pode alterar a sua configuração."

	# addresses
	m sec_addr "Enderezos" "Direcciones" "Addresses" "Adreces" "Endereços"
	m q_web "Dominio da aplicación web (p. ex. reservas.concello.gal). Ten que apuntar (DNS A/AAAA) a este servidor." \
		"Dominio de la aplicación web (p. ej. reservas.concello.gal). Tiene que apuntar (DNS A/AAAA) a este servidor." \
		"Web app domain (e.g. reservas.concello.gal). It must point (DNS A/AAAA) to this server." \
		"Domini de l'aplicació web (p. ex. reserves.ajuntament.cat). Ha d'apuntar (DNS A/AAAA) a aquest servidor." \
		"Domínio da aplicação web (p. ex. reservas.camara.pt). Tem de apontar (DNS A/AAAA) para este servidor."
	m q_auth "Dominio do inicio de sesión. Tamén ten que apuntar a este servidor." "Dominio del inicio de sesión. También tiene que apuntar a este servidor." \
		"Sign-in domain. It must point to this server too." "Domini de l'inici de sessió. També ha d'apuntar a aquest servidor." "Domínio do início de sessão. Também tem de apontar para este servidor."
	m dns_warn "Estes nomes aínda non resolven. Apúntaos a este servidor antes de que a xente use LibreQuenda:" \
		"Estos nombres todavía no resuelven. Apúntalos a este servidor antes de que la gente use LibreQuenda:" \
		"These names do not resolve yet. Point them to this server before people use LibreQuenda:" \
		"Aquests noms encara no resolen. Apunta'ls a aquest servidor abans que la gent faci servir LibreQuenda:" \
		"Estes nomes ainda não resolvem. Aponte-os para este servidor antes de as pessoas usarem o LibreQuenda:"

	# network
	m sec_net "Rede e portos" "Red y puertos" "Network and ports" "Xarxa i ports" "Rede e portas"
	m net_q "Como chega o tráfico de internet a LibreQuenda?" "¿Cómo llega el tráfico de internet a LibreQuenda?" \
		"How does traffic from the internet reach LibreQuenda?" "Com arriba el trànsit d'internet a LibreQuenda?" "Como chega o tráfego da internet ao LibreQuenda?"
	m net_direct "Directamente: portos 80 e 443, con certificados de Let's Encrypt" \
		"Directamente: puertos 80 y 443, con certificados de Let's Encrypt" \
		"Directly: ports 80 and 443, with Let's Encrypt certificates" \
		"Directament: ports 80 i 443, amb certificats de Let's Encrypt" \
		"Diretamente: portas 80 e 443, com certificados da Let's Encrypt"
	m net_proxy "Detrás doutro servidor web que xa fai HTTPS (nginx, Apache…)" \
		"Detrás de otro servidor web que ya hace HTTPS (nginx, Apache…)" \
		"Behind another web server that already does HTTPS (nginx, Apache…)" \
		"Darrere d'un altre servidor web que ja fa HTTPS (nginx, Apache…)" \
		"Atrás de outro servidor web que já faz HTTPS (nginx, Apache…)"
	m net_busy "O porto %s xa está en uso%s." "El puerto %s ya está en uso%s." "Port %s is already in use%s." "El port %s ja està en ús%s." "A porta %s já está em uso%s."
	m net_busy_direct "LibreQuenda non poderá arrincar mentres estea ocupado. Se xa tes un servidor web, escolle «detrás doutro servidor web». Continuar igualmente?" \
		"LibreQuenda no podrá arrancar mientras esté ocupado. Si ya tienes un servidor web, elige «detrás de otro servidor web». ¿Continuar igualmente?" \
		"LibreQuenda cannot start while it is taken. If you already have a web server, choose «behind another web server». Continue anyway?" \
		"LibreQuenda no podrà arrencar mentre estigui ocupat. Si ja tens un servidor web, tria «darrere d'un altre servidor web». Vols continuar igualment?" \
		"O LibreQuenda não poderá arrancar enquanto estiver ocupada. Se já tem um servidor web, escolha «atrás de outro servidor web». Continuar mesmo assim?"
	m q_port "Porto no que LibreQuenda lle serve HTTP ao teu servidor web:" "Puerto en el que LibreQuenda le sirve HTTP a tu servidor web:" \
		"Port on which LibreQuenda serves HTTP to your web server:" "Port on LibreQuenda serveix HTTP al teu servidor web:" "Porta em que o LibreQuenda serve HTTP ao seu servidor web:"
	m q_reach "Onde está ese servidor web?" "¿Dónde está ese servidor web?" "Where is that web server?" "On és aquest servidor web?" "Onde está esse servidor web?"
	m reach_local "Neste mesmo servidor (só 127.0.0.1)" "En este mismo servidor (solo 127.0.0.1)" "On this same server (127.0.0.1 only)" "En aquest mateix servidor (només 127.0.0.1)" "Neste mesmo servidor (só 127.0.0.1)"
	m reach_any "Noutra máquina (todas as interfaces)" "En otra máquina (todas las interfaces)" "On another machine (all interfaces)" "En una altra màquina (totes les interfícies)" "Noutra máquina (todas as interfaces)"
	m reach_any_warn "LibreQuenda servirá HTTP sen cifrar no porto %s en todas as interfaces: no firewall, ábreo só para o teu servidor web." \
		"LibreQuenda servirá HTTP sin cifrar en el puerto %s en todas las interfaces: en el firewall, ábrelo solo para tu servidor web." \
		"LibreQuenda will serve unencrypted HTTP on port %s on every interface: in the firewall, open it to your web server only." \
		"LibreQuenda servirà HTTP sense xifrar al port %s a totes les interfícies: al tallafoc, obre'l només per al teu servidor web." \
		"O LibreQuenda vai servir HTTP sem cifrar na porta %s em todas as interfaces: na firewall, abra-a só para o seu servidor web."
	m sum_direct "directo, 80 e 443" "directo, 80 y 443" "direct, 80 and 443" "directe, 80 i 443" "direto, 80 e 443"
	m sum_proxy "detrás do teu servidor web → %s" "detrás de tu servidor web → %s" "behind your web server → %s" "darrere del teu servidor web → %s" "atrás do seu servidor web → %s"

	# administrator
	m sec_admin "Primeira persoa administradora" "Primera persona administradora" "First administrator" "Primera persona administradora" "Primeiro administrador"
	m admin_note "Inicia sesión para crear os concellos (en /admin) e xestiona o servizo de inicio de sesión." \
		"Inicia sesión para crear los ayuntamientos (en /admin) y gestiona el servicio de inicio de sesión." \
		"Signs in to create the councils (at /admin) and manages the sign-in service." \
		"Inicia la sessió per crear els ajuntaments (a /admin) i gestiona el servei d'inici de sessió." \
		"Inicia sessão para criar os municípios (em /admin) e gere o serviço de início de sessão."
	m q_email "Correo electrónico" "Correo electrónico" "Email" "Correu electrònic" "Email"
	m q_first "Nome" "Nombre" "First name" "Nom" "Nome"
	m q_last "Apelidos" "Apellidos" "Last name" "Cognoms" "Apelidos"
	m q_admin_pw "Contrasinal (8 caracteres ou máis, con maiúsculas, minúsculas, un número e un símbolo)" \
		"Contraseña (8 caracteres o más, con mayúsculas, minúsculas, un número y un símbolo)" \
		"Password (8 characters or more, with upper and lower case, a number and a symbol)" \
		"Contrasenya (8 caràcters o més, amb majúscules, minúscules, un número i un símbol)" \
		"Palavra-passe (8 caracteres ou mais, com maiúsculas, minúsculas, um número e um símbolo)"
	m empty_means "Baleiro: %s." "Vacío: %s." "Empty: %s." "Buit: %s." "Vazio: %s."
	m hint_keep "consérvase o actual" "se conserva la actual" "the current one stays" "es conserva l'actual" "mantém-se a atual"
	m hint_gen "xérase un e móstrase ao final" "se genera una y se muestra al final" "one is generated and shown at the end" "se'n genera una i es mostra al final" "é gerada uma e mostrada no fim"
	m hint_none "sen contrasinal" "sin contraseña" "no password" "sense contrasenya" "sem palavra-passe"
	m q_pw_again "Repíteo:" "Repítela:" "Again:" "Repeteix-la:" "Repita-a:"
	m pw_mismatch "Non coinciden." "No coinciden." "They do not match." "No coincideixen." "Não coincidem."

	# email
	m sec_smtp "Correo (SMTP)" "Correo (SMTP)" "Email (SMTP)" "Correu (SMTP)" "Email (SMTP)"
	m smtp_note "Por el saen as confirmacións de reserva, os recordatorios, os códigos de inicio de sesión e os restablecementos de contrasinal." \
		"Por él salen las confirmaciones de reserva, los recordatorios, los códigos de inicio de sesión y los restablecimientos de contraseña." \
		"Booking confirmations, reminders, sign-in codes and password resets go out through it." \
		"Hi surten les confirmacions de reserva, els recordatoris, els codis d'inici de sessió i els restabliments de contrasenya." \
		"Por ele saem as confirmações de reserva, os lembretes, os códigos de início de sessão e as reposições de palavra-passe."
	m q_smtp_host "Servidor SMTP (p. ex. smtp.concello.gal, smtp.gmail.com)" "Servidor SMTP (p. ej. smtp.concello.gal, smtp.gmail.com)" \
		"SMTP server (e.g. smtp.concello.gal, smtp.gmail.com)" "Servidor SMTP (p. ex. smtp.ajuntament.cat, smtp.gmail.com)" "Servidor SMTP (p. ex. smtp.camara.pt, smtp.gmail.com)"
	m q_smtp_port "Porto e cifrado do servidor SMTP:" "Puerto y cifrado del servidor SMTP:" "The SMTP server's port and encryption:" "Port i xifratge del servidor SMTP:" "Porta e cifra do servidor SMTP:"
	m other_port "Outro porto" "Otro puerto" "Another port" "Un altre port" "Outra porta"
	m q_port_n "Porto" "Puerto" "Port" "Port" "Porta"
	m q_smtp_user "Usuario (adoita ser o enderezo completo; baleiro se o servidor non pide autenticación)" \
		"Usuario (suele ser la dirección completa; vacío si el servidor no pide autenticación)" \
		"Username (usually the full address; empty if the server needs no authentication)" \
		"Usuari (sol ser l'adreça completa; buit si el servidor no demana autenticació)" \
		"Utilizador (normalmente o endereço completo; vazio se o servidor não pede autenticação)"
	m q_smtp_pw "Contrasinal SMTP (con Gmail, un contrasinal de aplicación: myaccount.google.com/apppasswords)" \
		"Contraseña SMTP (con Gmail, una contraseña de aplicación: myaccount.google.com/apppasswords)" \
		"SMTP password (with Gmail, an app password: myaccount.google.com/apppasswords)" \
		"Contrasenya SMTP (amb Gmail, una contrasenya d'aplicació: myaccount.google.com/apppasswords)" \
		"Palavra-passe SMTP (com o Gmail, uma palavra-passe de aplicação: myaccount.google.com/apppasswords)"
	m q_from "Enderezo remitente (Gmail e Microsoft 365 só envían desde o da propia conta)" \
		"Dirección remitente (Gmail y Microsoft 365 solo envían desde la de la propia cuenta)" \
		"Sender address (Gmail and Microsoft 365 only send from the account's own)" \
		"Adreça remitent (Gmail i Microsoft 365 només envien des de la del compte mateix)" \
		"Endereço do remetente (o Gmail e o Microsoft 365 só enviam a partir do da própria conta)"
	m q_sender "Nome do remitente" "Nombre del remitente" "Sender name" "Nom del remitent" "Nome do remetente"

	# support
	m sec_support "Asistencia (opcional)" "Asistencia (opcional)" "Support (optional)" "Assistència (opcional)" "Apoio (opcional)"
	m support_note "Móstrase na app e na web para quen precise axuda. Baleiro: non se mostra." \
		"Se muestra en la app y en la web para quien necesite ayuda. Vacío: no se muestra." \
		"Shown in the app and the web for people who need help. Empty: not shown." \
		"Es mostra a l'app i al web per a qui necessiti ajuda. Buit: no es mostra." \
		"Mostrado na app e na web para quem precisar de ajuda. Vazio: não é mostrado."
	m q_sup_email "Correo de asistencia" "Correo de asistencia" "Support email" "Correu d'assistència" "Email de apoio"
	m q_sup_phone "Teléfono de asistencia" "Teléfono de asistencia" "Support phone" "Telèfon d'assistència" "Telefone de apoio"

	# android app
	m sec_app "A túa propia app Android" "Tu propia app Android" "Your own Android app" "La teva pròpia app Android" "A sua própria app Android"
	m app_note "A app de LibreQuenda pódese compilar co teu nome, logotipo e paquete, xa apuntando a este servidor: un APK para instalar directamente e un AAB para Google Play. A compilación precisa uns 4 GB de RAM." \
		"La app de LibreQuenda se puede compilar con tu nombre, logotipo y paquete, ya apuntando a este servidor: un APK para instalar directamente y un AAB para Google Play. La compilación necesita unos 4 GB de RAM." \
		"LibreQuenda's app can be built with your name, logo and package, already pointing to this server: an APK to install directly and an AAB for Google Play. The build needs about 4 GB of RAM." \
		"L'app de LibreQuenda es pot compilar amb el teu nom, logotip i paquet, ja apuntant a aquest servidor: un APK per instal·lar directament i un AAB per a Google Play. La compilació necessita uns 4 GB de RAM." \
		"A app do LibreQuenda pode ser compilada com o seu nome, logótipo e pacote, já a apontar para este servidor: um APK para instalar diretamente e um AAB para o Google Play. A compilação precisa de cerca de 4 GB de RAM."
	m q_app_now "Compilar agora a túa app?" "¿Compilar ahora tu app?" "Build your own app now?" "Vols compilar ara la teva app?" "Compilar agora a sua app?"
	m q_app_web "Dominio web do teu LibreQuenda (p. ex. reservas.concello.gal)" "Dominio web de tu LibreQuenda (p. ej. reservas.concello.gal)" \
		"Your LibreQuenda's web domain (e.g. reservas.concello.gal)" "Domini web del teu LibreQuenda (p. ex. reserves.ajuntament.cat)" "Domínio web do seu LibreQuenda (p. ex. reservas.camara.pt)"
	m q_app_name "Nome da app (debaixo da icona)" "Nombre de la app (debajo del icono)" "App name (under the icon)" "Nom de l'app (sota la icona)" "Nome da app (por baixo do ícone)"
	m app_name_def "Reservas" "Reservas" "Bookings" "Reserves" "Reservas"
	m q_app_id "Paquete (applicationId). Identifica a app nos teléfonos e en Google Play, e non pode cambiar unha vez publicada. O inicio de sesión volve á app a través del, así que o servidor o lista (ANDROID_APP_IDS)." \
		"Paquete (applicationId). Identifica la app en los teléfonos y en Google Play, y no puede cambiar una vez publicada. El inicio de sesión vuelve a la app a través de él, así que el servidor lo lista (ANDROID_APP_IDS)." \
		"Package (applicationId). It identifies the app on phones and on Google Play, and can never change once published. Sign-in returns to the app through it, so the server lists it (ANDROID_APP_IDS)." \
		"Paquet (applicationId). Identifica l'app als telèfons i a Google Play, i no pot canviar un cop publicada. L'inici de sessió torna a l'app a través d'ell, així que el servidor el llista (ANDROID_APP_IDS)." \
		"Pacote (applicationId). Identifica a app nos telemóveis e no Google Play, e não pode mudar depois de publicada. O início de sessão volta à app através dele, por isso o servidor lista-o (ANDROID_APP_IDS)."
	m q_logo "Ficheiro do logotipo: cadrado, de 512×512 ou máis (mellor PNG con transparencia). Baleiro: a icona de LibreQuenda." \
		"Archivo del logotipo: cuadrado, de 512×512 o más (mejor PNG con transparencia). Vacío: el icono de LibreQuenda." \
		"Logo file: square, 512×512 or larger (PNG with transparency is best). Empty: LibreQuenda's icon." \
		"Fitxer del logotip: quadrat, de 512×512 o més (millor PNG amb transparència). Buit: la icona de LibreQuenda." \
		"Ficheiro do logótipo: quadrado, de 512×512 ou mais (de preferência PNG com transparência). Vazio: o ícone do LibreQuenda."
	m q_bg "Cor de fondo da icona" "Color de fondo del icono" "Icon background colour" "Color de fons de la icona" "Cor de fundo do ícone"
	m app_confirm "Compilar %s (%s) para %s?" "¿Compilar %s (%s) para %s?" "Build %s (%s) for %s?" "Vols compilar %s (%s) per a %s?" "Compilar %s (%s) para %s?"

	# review
	m review_title "Revisión" "Revisión" "Review" "Revisió" "Revisão"
	m review_q "Escolle unha sección para cambiala, ou instala." "Elige una sección para cambiarla, o instala." \
		"Choose a section to change it, or install." "Tria una secció per canviar-la, o instal·la." "Escolha uma secção para a alterar, ou instale."
	m review_go "» Gardar e arrincar LibreQuenda" "» Guardar y arrancar LibreQuenda" "» Save and start LibreQuenda" "» Desa i arrenca LibreQuenda" "» Guardar e arrancar o LibreQuenda"

	# validation
	m v_quote "Sen comiñas simples ('), por favor." "Sin comillas simples ('), por favor." "No single quotes ('), please." "Sense cometes simples ('), si us plau." "Sem plicas ('), por favor."
	m v_domain "Ten que ser un dominio como reservas.concello.gal." "Tiene que ser un dominio como reservas.concello.gal." "It must be a domain like reservas.concello.gal." "Ha de ser un domini com reserves.ajuntament.cat." "Tem de ser um domínio como reservas.camara.pt."
	m v_email "Ten que ser un enderezo de correo." "Tiene que ser una dirección de correo." "It must be an email address." "Ha de ser una adreça de correu." "Tem de ser um endereço de email."
	m v_nonempty "Este campo é obrigatorio." "Este campo es obligatorio." "This field is required." "Aquest camp és obligatori." "Este campo é obrigatório."
	m v_port "Ten que ser un número de porto (1–65535)." "Tiene que ser un número de puerto (1–65535)." "It must be a port number (1–65535)." "Ha de ser un número de port (1–65535)." "Tem de ser um número de porta (1–65535)."
	m v_password "Polo menos 8 caracteres, con maiúsculas e minúsculas, un número e un símbolo." "Al menos 8 caracteres, con mayúsculas y minúsculas, un número y un símbolo." \
		"At least 8 characters, with upper and lower case, a number and a symbol." "Almenys 8 caràcters, amb majúscules i minúscules, un número i un símbol." "Pelo menos 8 caracteres, com maiúsculas e minúsculas, um número e um símbolo."
	m v_app_id "Palabras en minúscula separadas por puntos, p. ex. gal.lalin.reservas." "Palabras en minúscula separadas por puntos, p. ej. gal.lalin.reservas." \
		"Lowercase words separated by dots, e.g. gal.lalin.reservas." "Paraules en minúscula separades per punts, p. ex. cat.olot.reserves." "Palavras em minúsculas separadas por pontos, p. ex. pt.braga.reservas."
	m v_app_id_own "Ese é o paquete da app de LibreQuenda: escolle o teu." "Ese es el paquete de la app de LibreQuenda: elige el tuyo." \
		"That one is LibreQuenda's own app: choose yours." "Aquest és el paquet de l'app de LibreQuenda: tria el teu." "Esse é o pacote da app do LibreQuenda: escolha o seu."
	m v_colour "Ten que ser unha cor como #FFFFFF." "Tiene que ser un color como #FFFFFF." "It must be a colour like #FFFFFF." "Ha de ser un color com #FFFFFF." "Tem de ser uma cor como #FFFFFF."
	m v_nofile "Non existe ese ficheiro." "No existe ese archivo." "There is no such file." "No existeix aquest fitxer." "Esse ficheiro não existe."
	m v_image "Ten que ser unha imaxe PNG, JPG ou WebP." "Tiene que ser una imagen PNG, JPG o WebP." "It must be a PNG, JPG or WebP image." "Ha de ser una imatge PNG, JPG o WebP." "Tem de ser uma imagem PNG, JPG ou WebP."

	# running
	m wrote_env "escribiuse %s" "se ha escrito %s" "wrote %s" "s'ha escrit %s" "foi escrito %s"
	m starting "Arrincando LibreQuenda" "Arrancando LibreQuenda" "Starting LibreQuenda" "Arrencant LibreQuenda" "A arrancar o LibreQuenda"
	m start_fail "Algo non arrincou: docker compose ps, e docker compose logs <servizo>, en %s" "Algo no ha arrancado: docker compose ps, y docker compose logs <servicio>, en %s" \
		"Something did not start: docker compose ps, and docker compose logs <service>, in %s" "Alguna cosa no ha arrencat: docker compose ps, i docker compose logs <servei>, a %s" \
		"Algo não arrancou: docker compose ps, e docker compose logs <serviço>, em %s"
	m running "en marcha en https://%s" "en marcha en https://%s" "running at https://%s" "en marxa a https://%s" "em execução em https://%s"
	m done "Feito" "Hecho" "Done" "Fet" "Concluído"
	m f_web "Aplicación web:" "Aplicación web:" "Web app:" "Aplicació web:" "Aplicação web:"
	m f_admin "Administración:" "Administración:" "Administration:" "Administració:" "Administração:"
	m f_as "(inicia sesión como %s)" "(inicia sesión como %s)" "(sign in as %s)" "(inicia la sessió com a %s)" "(inicie sessão como %s)"
	m f_signin "Inicio de sesión:" "Inicio de sesión:" "Sign-in admin:" "Inici de sessió:" "Início de sessão:"
	m f_pw "Contrasinal da persoa administradora: %s" "Contraseña de la persona administradora: %s" "Administrator password: %s" "Contrasenya de la persona administradora: %s" "Palavra-passe do administrador: %s"
	m f_keep "Garda unha copia de %s nun lugar seguro: ZITADEL_MASTERKEY non se pode recuperar, e sen ela non se poden restaurar as copias de seguridade." \
		"Guarda una copia de %s en un lugar seguro: ZITADEL_MASTERKEY no se puede recuperar, y sin ella no se pueden restaurar las copias de seguridad." \
		"Keep a copy of %s somewhere safe: ZITADEL_MASTERKEY cannot be recovered, and the backups cannot be restored without it." \
		"Desa una còpia de %s en un lloc segur: ZITADEL_MASTERKEY no es pot recuperar, i sense ella no es poden restaurar les còpies de seguretat." \
		"Guarde uma cópia de %s num lugar seguro: a ZITADEL_MASTERKEY não pode ser recuperada, e sem ela não é possível restaurar as cópias de segurança."
	m f_update "Para actualizar máis adiante: %s" "Para actualizar más adelante: %s" "Update later: %s" "Per actualitzar més endavant: %s" "Para atualizar mais tarde: %s"
	m f_proxy "LibreQuenda escoita en http://%s. Configura o teu servidor web para que lle envíe %s e %s, coa cabeceira Host orixinal e X-Forwarded-Proto: https." \
		"LibreQuenda escucha en http://%s. Configura tu servidor web para que le envíe %s y %s, con la cabecera Host original y X-Forwarded-Proto: https." \
		"LibreQuenda listens on http://%s. Set up your web server to send it %s and %s, with the original Host header and X-Forwarded-Proto: https." \
		"LibreQuenda escolta a http://%s. Configura el teu servidor web perquè li enviï %s i %s, amb la capçalera Host original i X-Forwarded-Proto: https." \
		"O LibreQuenda escuta em http://%s. Configure o seu servidor web para lhe enviar %s e %s, com o cabeçalho Host original e X-Forwarded-Proto: https."
	m f_nginx "Con nginx, a configuración xa está escrita en %s:" "Con nginx, la configuración ya está escrita en %s:" "With nginx, the configuration is already written in %s:" \
		"Amb nginx, la configuració ja està escrita a %s:" "Com o nginx, a configuração já está escrita em %s:"

	# android build
	m build_title "Compilando %s (%s)" "Compilando %s (%s)" "Building %s (%s)" "Compilant %s (%s)" "A compilar %s (%s)"
	m builder_prep "Preparando o entorno de compilación (a primeira vez leva uns minutos)…" "Preparando el entorno de compilación (la primera vez tarda unos minutos)…" \
		"Preparing the build environment (the first time takes a few minutes)…" "Preparant l'entorn de compilació (la primera vegada triga uns minuts)…" "A preparar o ambiente de compilação (da primeira vez demora alguns minutos)…"
	m builder_fail "Non se puido preparar o entorno de compilación" "No se ha podido preparar el entorno de compilación" "Could not prepare the build environment" \
		"No s'ha pogut preparar l'entorn de compilació" "Não foi possível preparar o ambiente de compilação"
	m key_create "Creando a chave de sinatura da app…" "Creando la clave de firma de la app…" "Creating the app's signing key…" "Creant la clau de signatura de l'app…" "A criar a chave de assinatura da app…"
	m key_fail "Non se puido crear a chave de sinatura" "No se ha podido crear la clave de firma" "Could not create the signing key" "No s'ha pogut crear la clau de signatura" "Não foi possível criar a chave de assinatura"
	m key_orphan "%s non ten %s ao lado" "%s no tiene %s al lado" "%s has no %s next to it" "%s no té %s al costat" "%s não tem %s ao lado"
	m building "Compilando (de 5 a 10 minutos a primeira vez)…" "Compilando (de 5 a 10 minutos la primera vez)…" "Building (5 to 10 minutes the first time)…" \
		"Compilant (de 5 a 10 minuts la primera vegada)…" "A compilar (5 a 10 minutos da primeira vez)…"
	m build_fail "A compilación fallou (a saída de enriba di por que)" "La compilación ha fallado (la salida de arriba dice por qué)" "The build failed (the output above says why)" \
		"La compilació ha fallat (la sortida de dalt diu per què)" "A compilação falhou (a saída acima diz porquê)"
	m built "compilados %s e %s" "compilados %s y %s" "built %s and %s" "compilats %s i %s" "compilados %s e %s"
	m apk_line "instálao directamente: envíao aos teléfonos ou publícao na túa web" "instálalo directamente: envíalo a los teléfonos o publícalo en tu web" \
		"install it directly: send it to phones, or publish it on your website" "instal·la'l directament: envia'l als telèfons o publica'l al teu web" "instale-o diretamente: envie-o para os telemóveis ou publique-o no seu site"
	m aab_line "súbeo a Google Play (Play Console → a túa app → nova versión)" "súbelo a Google Play (Play Console → tu app → nueva versión)" \
		"upload it to Google Play (Play Console → your app → a new release)" "puja'l a Google Play (Play Console → la teva app → nova versió)" "carregue-o no Google Play (Play Console → a sua app → nova versão)"
	m key_backup "Fai agora unha copia de %s e %s noutro lugar. Cada actualización da app ten que asinarse con esta chave: sen ela, os teléfonos e Google Play rexeitan as actualizacións e a app ten que publicarse de novo cun paquete novo." \
		"Haz ahora una copia de %s y %s en otro lugar. Cada actualización de la app tiene que firmarse con esta clave: sin ella, los teléfonos y Google Play rechazan las actualizaciones y la app tiene que publicarse de nuevo con un paquete nuevo." \
		"Back up %s and %s now, somewhere else. Every update of the app must be signed with this key: without it, phones and Google Play refuse updates and the app has to be published again under a new package." \
		"Fes ara una còpia de %s i %s en un altre lloc. Cada actualització de l'app s'ha de signar amb aquesta clau: sense ella, els telèfons i Google Play rebutgen les actualitzacions i l'app s'ha de tornar a publicar amb un paquet nou." \
		"Faça já uma cópia de %s e %s noutro lugar. Cada atualização da app tem de ser assinada com esta chave: sem ela, os telemóveis e o Google Play recusam as atualizações e a app tem de ser publicada de novo com um pacote novo."
	m app_update "Para actualizar a app máis adiante (despois de actualizar LibreQuenda): %s" "Para actualizar la app más adelante (después de actualizar LibreQuenda): %s" \
		"Update the app later (after updating LibreQuenda): %s" "Per actualitzar l'app més endavant (després d'actualitzar LibreQuenda): %s" "Para atualizar a app mais tarde (depois de atualizar o LibreQuenda): %s"
	m reg_missing "O servidor de LibreQuenda non está en %s. Para que o inicio de sesión funcione na app, engade o seu paquete ao .env do servidor e reinícialo alí:" \
		"El servidor de LibreQuenda no está en %s. Para que el inicio de sesión funcione en la app, añade su paquete al .env del servidor y reinícialo allí:" \
		"The LibreQuenda server is not in %s. For sign-in to work in the app, add its package to the server's .env and restart it there:" \
		"El servidor de LibreQuenda no és a %s. Perquè l'inici de sessió funcioni a l'app, afegeix-ne el paquet al .env del servidor i reinicia'l allà:" \
		"O servidor do LibreQuenda não está em %s. Para o início de sessão funcionar na app, adicione o pacote ao .env do servidor e reinicie-o lá:"
	m reg_comma "(separados por comas se hai varios)" "(separados por comas si hay varios)" "(comma-separated if there are several)" "(separats per comes si n'hi ha diversos)" "(separados por vírgulas se houver vários)"
	m reg_added "engadiuse %s a ANDROID_APP_IDS en %s" "se ha añadido %s a ANDROID_APP_IDS en %s" "added %s to ANDROID_APP_IDS in %s" "s'ha afegit %s a ANDROID_APP_IDS a %s" "%s foi adicionado a ANDROID_APP_IDS em %s"
	m chk_unreach "Non se puido acceder a %s para comprobar que o inicio de sesión acepta a app" "No se ha podido acceder a %s para comprobar que el inicio de sesión acepta la app" \
		"Could not reach %s to check that sign-in accepts the app" "No s'ha pogut accedir a %s per comprovar que l'inici de sessió accepta l'app" "Não foi possível aceder a %s para verificar que o início de sessão aceita a app"
	m chk_ok "o servidor acepta o inicio de sesión desde %s" "el servidor acepta el inicio de sesión desde %s" "the server accepts sign-in from %s" "el servidor accepta l'inici de sessió des de %s" "o servidor aceita o início de sessão a partir de %s"
	m chk_missing "O servidor aínda non lista %s, así que o inicio de sesión na app fallará." "El servidor todavía no lista %s, así que el inicio de sesión en la app fallará." \
		"The server does not list %s yet, so signing in to the app will fail." "El servidor encara no llista %s, així que l'inici de sessió a l'app fallarà." "O servidor ainda não lista %s, por isso o início de sessão na app vai falhar."
	m chk_revoked "ZITADEL_SETUP_REVOKE_TOKEN=true: a configuración xa non pode cambiar Zitadel. Crea un token novo para librequenda-setup (docs/self-hosting.md), e despois: docker compose up -d --wait" \
		"ZITADEL_SETUP_REVOKE_TOKEN=true: la configuración ya no puede cambiar Zitadel. Crea un token nuevo para librequenda-setup (docs/self-hosting.md), y después: docker compose up -d --wait" \
		"ZITADEL_SETUP_REVOKE_TOKEN=true: the setup can no longer change Zitadel. Create a new token for librequenda-setup (docs/self-hosting.md), then: docker compose up -d --wait" \
		"ZITADEL_SETUP_REVOKE_TOKEN=true: la configuració ja no pot canviar Zitadel. Crea un token nou per a librequenda-setup (docs/self-hosting.md), i després: docker compose up -d --wait" \
		"ZITADEL_SETUP_REVOKE_TOKEN=true: a configuração já não pode alterar o Zitadel. Crie um token novo para librequenda-setup (docs/self-hosting.md), e depois: docker compose up -d --wait"
	m chk_logs "Comproba: docker compose logs zitadel-setup" "Comprueba: docker compose logs zitadel-setup" "Check: docker compose logs zitadel-setup" "Comprova: docker compose logs zitadel-setup" "Verifique: docker compose logs zitadel-setup"
}

# ============================================================
#  Screens (whiptail)
# ============================================================
TITLE=LibreQuenda
W=76 SH=22

screen_size() {
	local r=0 c=0
	read -r r c < <(stty size </dev/tty 2>/dev/null || echo "24 80") || true
	[ "${c:-0}" -ge 40 ] || c=80
	[ "${r:-0}" -ge 12 ] || r=24
	W=$((c - 4)); [ $W -le 76 ] || W=76
	SH=$((r - 2))
}

box_h() { # box_h "text" extra: rows for the text wrapped in the box, plus extra, within the screen
	local line n=0 w=$((W - 6))
	while IFS= read -r line; do
		if [ -z "$line" ]; then n=$((n + 1)); else n=$((n + (${#line} + w - 1) / w)); fi
	done <<<"$1"
	n=$((n + $2))
	[ $n -le "$SH" ] || n=$SH
	printf '%s' "$n"
}

# the answer comes out on fd 4; the boxes are drawn on, and read from, the terminal
wt_raw() { whiptail --output-fd 4 "$@" 4>&1 1>/dev/tty 2>/dev/tty </dev/tty; }
wt() {
	wt_raw --backtitle "$(t backtitle)" --title " $TITLE " --yes-button "$(t yes)" --no-button "$(t no)" \
		--ok-button "$(t ok)" --cancel-button "$(t cancel)" "$@"
}

ui_quit() { # Cancel or Esc: leave, or back to the same question
	if wt --defaultno --yesno "$(t quit_q)" 8 "$W"; then
		clear; say "$(t quit_done)"; exit 0
	fi
}
ui_msg() { wt --msgbox "$1" "$(box_h "$1" 7)" "$W" || true; }
ui_input() { # ui_input "text" "initial" -> REPLY
	local out
	while ! out=$(wt --inputbox "$1" "$(box_h "$1" 8)" "$W" "$2"); do ui_quit; done
	REPLY=$out
}
ui_password() { # ui_password "text" -> REPLY
	local out
	while ! out=$(wt --passwordbox "$1" "$(box_h "$1" 8)" "$W"); do ui_quit; done
	REPLY=$out
}
ui_menu() { # ui_menu "text" default tag label [tag label…] -> REPLY
	local text=$1 def=$2 out; shift 2
	local n=$(($# / 2))
	while ! out=$(wt --notags --default-item "$def" --menu "$text" "$(box_h "$text" $((n + 8)))" "$W" "$n" "$@"); do ui_quit; done
	REPLY=$out
}
ui_yesno() { # ui_yesno "text" y|n
	local rc opts=()
	[ "$2" = y ] || opts=(--defaultno)
	while true; do
		rc=0; wt "${opts[@]}" --yesno "$1" "$(box_h "$1" 7)" "$W" || rc=$?
		case $rc in 0) return 0 ;; 1) return 1 ;; *) ui_quit ;; esac
	done
}

pick_language() {
	command -v whiptail >/dev/null 2>&1 || die "LibreQuenda's installer needs whiptail: sudo apt install whiptail (Debian, Ubuntu) · sudo dnf install newt (Fedora)"
	local l=${LIBREQUENDA_LANG:-}
	if [ -z "$l" ]; then
		case $LOCALE_LANG in gl*) l=gl ;; es*) l=es ;; ca*) l=ca ;; pt*) l=pt ;; *) l=en ;; esac
		l=$(wt_raw --title " LibreQuenda " --notags --default-item "$l" --ok-button OK --nocancel \
			--menu "Lingua · Idioma · Language · Llengua · Língua" 13 54 5 \
			gl Galego es Español en English ca Català pt Português) || { clear; exit 0; }
	fi
	case $l in gl) LI=0 ;; es) LI=1 ;; en) LI=2 ;; ca) LI=3 ;; pt) LI=4 ;; *) die "LIBREQUENDA_LANG: gl, es, en, ca, pt" ;; esac
	messages
}

# ============================================================
#  The .env, as a map
# ============================================================
declare -A ENV=()

env_load() { # KEY=VALUE lines; one layer of quotes removed
	local line key value
	while IFS= read -r line || [ -n "$line" ]; do
		[[ $line =~ ^([A-Z0-9_]+)=(.*)$ ]] || continue
		key=${BASH_REMATCH[1]} value=${BASH_REMATCH[2]}
		if [[ $value =~ ^\'(.*)\'$ || $value =~ ^\"(.*)\"$ ]]; then value=${BASH_REMATCH[1]}; fi
		ENV[$key]=$value
	done <"$1"
}

env_quote() { # unquoted when plain; single quotes (no interpolation in Compose) otherwise
	if [[ $1 =~ ^[A-Za-z0-9._@:/+=,-]*$ ]]; then printf '%s' "$1"; else printf "'%s'" "$1"; fi
}

env_write() { # .env.example's layout and comments, with our values; keys it lacks go last
	local example=$1 out=$2 line key
	declare -A seen=()
	{
		while IFS= read -r line || [ -n "$line" ]; do
			if [[ $line =~ ^([A-Z0-9_]+)= ]] && [ -n "${ENV[${BASH_REMATCH[1]}]+set}" ]; then
				key=${BASH_REMATCH[1]}; seen[$key]=1
				printf '%s=%s\n' "$key" "$(env_quote "${ENV[$key]}")"
			else
				printf '%s\n' "$line"
			fi
		done <"$example"
		for key in "${!ENV[@]}"; do
			[ -n "${seen[$key]+set}" ] || printf '%s=%s\n' "$key" "$(env_quote "${ENV[$key]}")"
		done
	} >"$out.tmp"
	chmod 600 "$out.tmp"
	mv "$out.tmp" "$out"
}

# ============================================================
#  Questions
# ============================================================
trim() { local s=$1; s=${s#"${s%%[![:space:]]*}"}; s=${s%"${s##*[![:space:]]}"}; printf '%s' "$s"; }

# ask KEY "text" [default] [validator]: the field starts with the current value, or the default
ask() {
	local key=$1 text=$2 def=${3:-} check=${4:-} answer
	answer=${ENV[$key]:-$def}
	while true; do
		ui_input "$text" "$answer"
		answer=$(trim "$REPLY")
		if [[ $answer == *"'"* ]]; then ui_msg "$(t v_quote)"; continue; fi
		if [ -n "$check" ] && ! "$check" "$answer"; then ui_msg "$ERR"; continue; fi
		ENV[$key]=$answer
		return
	done
}

# ask_secret KEY "text" generator [validator] [confirm]: hidden; empty keeps the current one or generates one
ask_secret() {
	local key=$1 text=$2 gen=$3 check=${4:-} confirm=${5:-} answer hint
	if [ -n "${ENV[$key]:-}" ]; then hint=$(t hint_keep)
	elif [ "$gen" = true ]; then hint=$(t hint_none)
	else hint=$(t hint_gen); fi
	while true; do
		ui_password "$text"$'\n\n'"$(t empty_means "$hint")"
		answer=$REPLY
		if [ -z "$answer" ]; then
			[ -n "${ENV[$key]:-}" ] && return
			ENV[$key]=$($gen)
			[ "$gen" = true ] || GENERATED+=("$key")
			return
		fi
		if [[ $answer == *"'"* ]]; then ui_msg "$(t v_quote)"; continue; fi
		if [ -n "$check" ] && ! "$check" "$answer"; then ui_msg "$ERR"; continue; fi
		if [ -n "$confirm" ]; then
			ui_password "$(t q_pw_again)"
			[ "$REPLY" = "$answer" ] || { ui_msg "$(t pw_mismatch)"; continue; }
		fi
		ENV[$key]=$answer
		return
	done
}

# ---------- validators: ERR says why ----------
ERR=""
bad() { ERR=$(t "$1"); return 1; }
v_domain() { [[ $1 =~ ^([a-z0-9]([a-z0-9-]*[a-z0-9])?\.)+[a-z]{2,}$ ]] || bad v_domain; }
v_email() { [[ $1 =~ ^[^@[:space:]]+@[^@[:space:]]+\.[^@[:space:]]+$ ]] || bad v_email; }
v_email_opt() { [ -z "$1" ] || v_email "$1"; }
v_nonempty() { [ -n "$1" ] || bad v_nonempty; }
v_port() { { [[ $1 =~ ^[0-9]+$ ]] && [ "$1" -ge 1 ] && [ "$1" -le 65535 ]; } || bad v_port; }
v_free_port() {
	v_port "$1" || return 1
	local o; o=$(port_owner "$1")
	[ -z "$o" ] || { ERR=$(t net_busy "$1" "$(owner_txt "$o")"); return 1; }
}
v_password() { # Zitadel's default policy: 8+, upper, lower, number, symbol
	{ [ ${#1} -ge 8 ] && [[ $1 =~ [A-Z] && $1 =~ [a-z] && $1 =~ [0-9] && $1 =~ [^A-Za-z0-9] ]]; } || bad v_password
}
v_app_id() {
	[[ $1 =~ ^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$ ]] || bad v_app_id || return 1
	[ "$1" != gal.tribo.librequenda ] || bad v_app_id_own
}
v_colour() { [[ $1 =~ ^#[0-9A-Fa-f]{6}$ ]] || bad v_colour; }
abs_path() { # a path typed by the person: ~ and relative to where the script was started
	local p=$1
	[[ $p == "~/"* ]] && p="$HOME/${p#\~/}"
	[[ $p == /* ]] || p="$START_PWD/$p"
	printf '%s' "$p"
}
v_logo() {
	[ -z "$1" ] && return 0
	[ -f "$(abs_path "$1")" ] || bad v_nofile || return 1
	[[ ${1,,} =~ \.(png|jpe?g|webp)$ ]] || bad v_image
}

# ---------- ports ----------
port_owner() { # who listens on a TCP port ("" when free, "-" when unknown); our own proxy does not count
	local p=$1 line name
	command -v ss >/dev/null 2>&1 || return 0
	line=$(ss -Htlnp "sport = :$p" 2>/dev/null | head -n1) || true
	[ -n "$line" ] || return 0
	if docker ps --filter "label=com.docker.compose.project.working_dir=$DIR" \
		--filter label=com.docker.compose.service=proxy --format '{{.Ports}}' 2>/dev/null | grep -Eq ":$p->"; then
		return 0
	fi
	name=$(grep -o 'users:(("[^"]*"' <<<"$line" | cut -d'"' -f2) || true
	printf '%s' "${name:--}"
}
owner_txt() { [ "$1" = - ] || printf ' (%s)' "$1"; }
free_port() { # free_port FROM [EXCEPT]: the first free port from FROM on
	local p=$1
	while [ "$p" = "${2:-}" ] || [ -n "$(port_owner "$p")" ]; do p=$((p + 1)); done
	printf '%s' "$p"
}

# ---------- secrets ----------
rand_alnum() { # no "tr </dev/urandom | head": tr dies of SIGPIPE, and pipefail stops everything
	local n=${1:-32} out=""
	while [ ${#out} -lt "$n" ]; do out+=$(head -c 64 /dev/urandom | LC_ALL=C tr -dc A-Za-z0-9); done
	printf '%s' "${out:0:n}"
}
rand_hex() { od -An -tx1 -N"$1" /dev/urandom | tr -d ' \n'; }
rand_b64() { head -c 32 /dev/urandom | base64 | tr -d '\n'; }
gen_alnum32() { rand_alnum 32; }
gen_hex32() { rand_hex 32; }
gen_storage_key() { printf 'GK%s' "$(rand_hex 12)"; }
gen_password() { printf '%s-Aa1' "$(rand_alnum 20)"; }
GENERATED=()

secret() { # secret KEY generator: generated once, then kept for good
	[ -n "${ENV[$1]:-}" ] || ENV[$1]=$($2)
}

# ---------- files from the repository ----------
fetch() { # fetch <path in the repo> <destination>
	if [ -n "$SOURCE" ]; then cp "$SOURCE/$1" "$2"
	else curl -fsSL "https://raw.githubusercontent.com/$REPO/$REF/$1" -o "$2" || die "$(t dl_fail "$1" "$REF")"; fi
}

source_tree() { # the whole repository (for the app), in $DIR/.source
	if [ -n "$SOURCE" ]; then SRC=$(cd "$SOURCE" && pwd); return; fi
	SRC="$DIR/.source"
	say "$(t src_dl "$REF")"
	rm -rf "$SRC"; mkdir -p "$SRC"
	curl -fsSL "https://codeload.github.com/$REPO/tar.gz/$REF" | tar -xz -C "$SRC" --strip-components=1 \
		|| die "$(t dl_fail source "$REF")"
}

need() { command -v "$1" >/dev/null 2>&1 || die "$(t need_cmd "$1" "$2")"; }

check_docker() {
	need docker "https://docs.docker.com/engine/install/"
	docker compose version >/dev/null 2>&1 || die "$(t need_compose)"
	docker info >/dev/null 2>&1 || die "$(t need_docker)"
}

# ============================================================
#  Server
# ============================================================
section_addresses() {
	TITLE=$(t sec_addr)
	ask WEB_DOMAIN "$(t q_web)" "" v_domain
	ask ZITADEL_DOMAIN "$(t q_auth)" "auth.${ENV[WEB_DOMAIN]}" v_domain
	local d missing=""
	for d in "${ENV[WEB_DOMAIN]}" "${ENV[ZITADEL_DOMAIN]}"; do
		getent hosts "$d" >/dev/null 2>&1 || missing+=$'\n'"  $d"
	done
	[ -z "$missing" ] || ui_msg "$(t dns_warn)"$'\n'"$missing"
}

section_network() {
	TITLE=$(t sec_net)
	local o80 o443 busy="" def text port bind https
	o80=$(port_owner 80); o443=$(port_owner 443)
	[ -z "$o80" ] || busy+=$(t net_busy 80 "$(owner_txt "$o80")")$'\n'
	[ -z "$o443" ] || busy+=$(t net_busy 443 "$(owner_txt "$o443")")$'\n'
	def=direct
	{ [ "${ENV[CADDY_AUTO_HTTPS]:-true}" = false ] || [ -n "$busy" ]; } && def=proxy
	text=$(t net_q); [ -z "$busy" ] || text+=$'\n\n'"$busy"
	ui_menu "$text" "$def" direct "$(t net_direct)" proxy "$(t net_proxy)"

	if [ "$REPLY" = direct ]; then
		if [ -n "$busy" ] && ! ui_yesno "$busy"$'\n'"$(t net_busy_direct)" n; then section_network; return; fi
		ENV[CADDY_AUTO_HTTPS]=true ENV[PROXY_HTTP_PORT]=80 ENV[PROXY_HTTPS_PORT]=443
		return
	fi

	ENV[CADDY_AUTO_HTTPS]=false
	local cur=${ENV[PROXY_HTTP_PORT]:-80} reach=local
	[[ $cur == *:* ]] || [ "$cur" = 80 ] || reach=any
	port=${cur##*:}
	[ "$port" != 80 ] || port=$(free_port 8080)
	ENV[_PORT]=$port
	ask _PORT "$(t q_port)" "" v_free_port
	port=${ENV[_PORT]}; unset 'ENV[_PORT]'
	ui_menu "$(t q_reach)" "$reach" local "$(t reach_local)" any "$(t reach_any)"
	if [ "$REPLY" = local ]; then bind=127.0.0.1:; else bind=""; ui_msg "$(t reach_any_warn "$port")"; fi
	ENV[PROXY_HTTP_PORT]=$bind$port
	# Caddy serves no HTTPS behind a proxy, but Compose still publishes the port: somewhere out of the way
	https=${ENV[PROXY_HTTPS_PORT]:-}; https=${https##*:}
	if ! [[ $https =~ ^[0-9]+$ ]] || [ "$https" = 443 ] || [ "$https" = "$port" ] || [ -n "$(port_owner "$https")" ]; then
		https=$(free_port 8443 "$port")
	fi
	ENV[PROXY_HTTPS_PORT]=127.0.0.1:$https
}

section_admin() {
	TITLE=$(t sec_admin)
	ask ZITADEL_ADMIN_EMAIL "$(t admin_note)"$'\n\n'"$(t q_email)" "" v_email
	ask ZITADEL_ADMIN_FIRSTNAME "$(t q_first)" "" v_nonempty
	ask ZITADEL_ADMIN_LASTNAME "$(t q_last)" "" v_nonempty
	ask_secret ZITADEL_ADMIN_PASSWORD "$(t q_admin_pw)" gen_password v_password confirm
}

section_smtp() {
	TITLE=$(t sec_smtp)
	ask SMTP_HOST "$(t smtp_note)"$'\n\n'"$(t q_smtp_host)" "" v_nonempty
	local sel=${ENV[SMTP_PORT]:-587}
	case $sel in 587|465) ;; *) sel=other ;; esac
	ui_menu "$(t q_smtp_port)" "$sel" 587 "587 · STARTTLS" 465 "465 · TLS" other "$(t other_port)"
	if [ "$REPLY" = other ]; then ask SMTP_PORT "$(t q_port_n)" "" v_port; else ENV[SMTP_PORT]=$REPLY; fi
	ask SMTP_USERNAME "$(t q_smtp_user)"
	ask_secret SMTP_PASSWORD "$(t q_smtp_pw)" true
	local from_def="noreply@${ENV[WEB_DOMAIN]}"
	[[ ${ENV[SMTP_USERNAME]} == *@*.* ]] && from_def=${ENV[SMTP_USERNAME]}
	ask SMTP_FROM_ADDRESS "$(t q_from)" "$from_def" v_email
	local sender_name=${ENV[SMTP_FROM]:-}
	[[ $sender_name =~ ^(.*[^ ])\ *\<.*\>$ ]] && sender_name=${BASH_REMATCH[1]} || sender_name=""
	ENV[_SENDER]=$sender_name
	ask _SENDER "$(t q_sender)" "LibreQuenda" v_nonempty
	ENV[SMTP_FROM]="${ENV[_SENDER]} <${ENV[SMTP_FROM_ADDRESS]}>"
	unset 'ENV[_SENDER]'
	if [ "${ENV[SMTP_PORT]}" = 465 ]; then ENV[ZITADEL_SMTP_TLS]=true; else ENV[ZITADEL_SMTP_TLS]=false; fi
}

section_support() {
	TITLE=$(t sec_support)
	ask SUPPORT_EMAIL "$(t support_note)"$'\n\n'"$(t q_sup_email)" "" v_email_opt
	ask SUPPORT_PHONE "$(t q_sup_phone)"
}

section_app() {
	TITLE=$(t sec_app)
	if ui_yesno "$(t app_note)"$'\n\n'"$(t q_app_now)" "$WANT_APP"; then WANT_APP=y; app_questions; else WANT_APP=n; fi
}

review_server() {
	local net app smtp
	while true; do
		TITLE=$(t review_title)
		if [ "${ENV[CADDY_AUTO_HTTPS]}" = true ]; then net=$(t sum_direct); else net=$(t sum_proxy "${ENV[PROXY_HTTP_PORT]}"); fi
		if [ "$WANT_APP" = y ]; then app="${APP[APP_NAME]} · ${APP[APP_ID]}"; else app=$(t no); fi
		smtp="${ENV[SMTP_HOST]}:${ENV[SMTP_PORT]}${ENV[SMTP_USERNAME]:+ · ${ENV[SMTP_USERNAME]}}"
		ui_menu "$(t review_q)" go \
			addr "$(t sec_addr): ${ENV[WEB_DOMAIN]} · ${ENV[ZITADEL_DOMAIN]}" \
			net "$(t sec_net): $net" \
			admin "$(t sec_admin): ${ENV[ZITADEL_ADMIN_EMAIL]}" \
			smtp "$(t sec_smtp): $smtp" \
			support "$(t sec_support): ${ENV[SUPPORT_EMAIL]:-—}${ENV[SUPPORT_PHONE]:+ · ${ENV[SUPPORT_PHONE]}}" \
			app "$(t sec_app): $app" \
			go "$(t review_go)"
		case $REPLY in
			addr) section_addresses ;;
			net) section_network ;;
			admin) section_admin ;;
			smtp) section_smtp ;;
			support) section_support ;;
			app) section_app ;;
			go) return ;;
		esac
	done
}

add_app_id() { # adds $1 to ANDROID_APP_IDS
	local ids=${ENV[ANDROID_APP_IDS]:-}
	[[ ",$ids," == *",$1,"* ]] && return
	ENV[ANDROID_APP_IDS]=${ids:+$ids,}$1
}

start_stack() {
	title "$(t starting)"
	(cd "$DIR" && docker compose pull --quiet && docker compose up -d --wait) || {
		warn "$(t start_fail "$DIR")"
		exit 1
	}
	ok "$(t running "${ENV[WEB_DOMAIN]}")"
}

nginx_conf() { # an nginx site for LibreQuenda behind it, next to the .env
	local upstream=${ENV[PROXY_HTTP_PORT]} file="$DIR/nginx-librequenda.conf"
	[[ $upstream == *:* ]] || upstream="$(hostname -I 2>/dev/null | awk '{print $1}'):$upstream"
	cat >"$file" <<EOF
# LibreQuenda behind nginx (written by install.sh). Enable it, then let certbot add HTTPS:
#   sudo cp $file /etc/nginx/sites-available/librequenda
#   sudo ln -s /etc/nginx/sites-available/librequenda /etc/nginx/sites-enabled/
#   sudo nginx -t && sudo systemctl reload nginx
#   sudo certbot --nginx -d ${ENV[WEB_DOMAIN]} -d ${ENV[ZITADEL_DOMAIN]}
server {
	listen 80;
	listen [::]:80;
	server_name ${ENV[WEB_DOMAIN]} ${ENV[ZITADEL_DOMAIN]};

	client_max_body_size 25m;

	location / {
		proxy_pass http://$upstream;
		proxy_http_version 1.1;
		proxy_set_header Host \$host;
		proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
		proxy_set_header X-Forwarded-Proto https;
		proxy_buffering off;
		proxy_read_timeout 300s;
	}
}
EOF
	NGINX_CONF=$file
}

install_server() {
	check_docker
	need curl "$(t why_curl)"
	screen_size
	mkdir -p "$DIR"

	local fresh=y
	if [ -f "$DIR/.env" ]; then fresh=n; env_load "$DIR/.env"; fi
	local example="$DIR/.env.example"
	fetch .env.example "$example"
	# the example's defaults, under what is already set
	declare -A current=(); local k
	for k in "${!ENV[@]}"; do current[$k]=${ENV[$k]}; done
	env_load "$example"
	for k in "${!current[@]}"; do ENV[$k]=${current[$k]}; done
	# the example's placeholder addresses are not answers
	for k in WEB_DOMAIN ZITADEL_DOMAIN SMTP_FROM SMTP_FROM_ADDRESS; do
		[[ ${ENV[$k]:-} == *example.gal* ]] && ENV[$k]=""
	done

	TITLE=LibreQuenda
	if [ $fresh = y ]; then ui_msg "$(t new_install "$DIR")"$'\n\n'"$(t nav_hint)"
	else ui_msg "$(t found_install "$DIR")"$'\n\n'"$(t nav_hint)"; fi

	WANT_APP=n
	section_addresses
	section_network
	section_admin
	section_smtp
	section_support
	section_app
	review_server
	clear

	# everything else: generated once and never changed (the masterkey cannot change)
	secret POSTGRES_PASSWORD gen_alnum32
	secret APP_DB_PASSWORD gen_alnum32
	secret ANALYTICS_DB_PASSWORD gen_alnum32
	secret ZITADEL_MASTERKEY gen_alnum32
	secret LOGIN_SESSION_COOKIE_SECRET rand_b64
	secret ZITADEL_DB_PASSWORD gen_alnum32
	secret STORAGE_ACCESS_KEY gen_storage_key
	secret STORAGE_SECRET_KEY gen_hex32
	secret GARAGE_RPC_SECRET gen_hex32
	secret GARAGE_ADMIN_TOKEN rand_b64
	secret SECRETS_KEY gen_hex32
	if [ "$WANT_APP" = y ]; then APP[WEB_DOMAIN]=${ENV[WEB_DOMAIN]}; add_app_id "${APP[APP_ID]}"; fi

	fetch docker-compose.yaml "$DIR/docker-compose.yaml"
	fetch install.sh "$DIR/install.sh.new" && chmod +x "$DIR/install.sh.new" && mv "$DIR/install.sh.new" "$DIR/install.sh"
	[ -f "$DIR/.env" ] && cp "$DIR/.env" "$DIR/.env.backup-$(date +%Y%m%d-%H%M%S)"
	env_write "$example" "$DIR/.env"
	ok "$(t wrote_env "$DIR/.env")"
	NGINX_CONF=""
	[ "${ENV[CADDY_AUTO_HTTPS]}" = true ] || nginx_conf

	start_stack
	if [ "$WANT_APP" = y ]; then build_app; check_registered; fi

	title "$(t done)"
	say "$(t f_web)  https://${ENV[WEB_DOMAIN]}"
	say "$(t f_admin)  https://${ENV[WEB_DOMAIN]}/admin  $(t f_as "${ENV[ZITADEL_ADMIN_EMAIL]}")"
	say "$(t f_signin)  https://${ENV[ZITADEL_DOMAIN]}/ui/console"
	if [[ " ${GENERATED[*]} " == *" ZITADEL_ADMIN_PASSWORD "* ]]; then
		say ""; say "${bold}$(t f_pw "${ENV[ZITADEL_ADMIN_PASSWORD]}")${off}"
	fi
	if [ -n "$NGINX_CONF" ]; then
		say ""
		warn "$(t f_proxy "${ENV[PROXY_HTTP_PORT]}" "${ENV[WEB_DOMAIN]}" "${ENV[ZITADEL_DOMAIN]}")"
		say "$(t f_nginx "$NGINX_CONF")"
		say ""
		say "  sudo cp $NGINX_CONF /etc/nginx/sites-available/librequenda"
		say "  sudo ln -s /etc/nginx/sites-available/librequenda /etc/nginx/sites-enabled/"
		say "  sudo nginx -t && sudo systemctl reload nginx"
		say "  sudo certbot --nginx -d ${ENV[WEB_DOMAIN]} -d ${ENV[ZITADEL_DOMAIN]}"
	fi
	say ""
	warn "$(t f_keep "$DIR/.env")"
	note "$(t f_update "cd $DIR && docker compose pull && docker compose up -d --wait")"
}

# ============================================================
#  Android app
# ============================================================
declare -A APP=()

app_questions() {
	TITLE=$(t sec_app)
	APPDIR="$DIR/android"
	if [ -f "$APPDIR/app.conf" ] && [ -z "${APP[APP_ID]:-}" ]; then
		while IFS='=' read -r k v; do [ -z "$k" ] || APP[$k]=$v; done <"$APPDIR/app.conf"
	fi

	# the server it talks to: this installation's, or asked
	if [ -n "${ENV[WEB_DOMAIN]:-}" ]; then
		APP[WEB_DOMAIN]=${ENV[WEB_DOMAIN]}
	else
		ENV[_WEB]=${APP[WEB_DOMAIN]:-}
		ask _WEB "$(t q_app_web)" "" v_domain
		APP[WEB_DOMAIN]=${ENV[_WEB]}; unset 'ENV[_WEB]'
	fi

	local reversed; reversed=$(awk -F. '{for (i = NF; i > 0; i--) printf "%s%s", $i, (i > 1 ? "." : "")}' <<<"${APP[WEB_DOMAIN]}" \
		| tr '-' '_' | sed -E 's/(^|\.)([0-9])/\1n\2/g')
	ENV[_NAME]=${APP[APP_NAME]:-}; ask _NAME "$(t q_app_name)" "$(t app_name_def)" v_nonempty
	ENV[_ID]=${APP[APP_ID]:-}; ask _ID "$(t q_app_id)" "$reversed" v_app_id
	ENV[_LOGO]=${APP[LOGO]:-}; ask _LOGO "$(t q_logo)" "" v_logo
	ENV[_BG]=${APP[ICON_BACKGROUND]:-}; ask _BG "$(t q_bg)" "#FFFFFF" v_colour
	APP[APP_NAME]=${ENV[_NAME]} APP[APP_ID]=${ENV[_ID]} APP[ICON_BACKGROUND]=${ENV[_BG]}
	APP[LOGO]=${ENV[_LOGO]:+$(abs_path "${ENV[_LOGO]}")}
	unset 'ENV[_NAME]' 'ENV[_ID]' 'ENV[_LOGO]' 'ENV[_BG]'
}

app_save() {
	mkdir -p "$APPDIR"
	if [ -n "${APP[LOGO]}" ]; then # a copy next to the key, for the next build
		local ext=${APP[LOGO]##*.} copy
		copy="$APPDIR/logo.${ext,,}"
		[ "$(realpath "${APP[LOGO]}")" = "$(realpath -m "$copy")" ] || cp "${APP[LOGO]}" "$copy"
		APP[LOGO]=$copy
	fi
	{
		for k in WEB_DOMAIN APP_NAME APP_ID LOGO ICON_BACKGROUND; do printf '%s=%s\n' "$k" "${APP[$k]}"; done
	} >"$APPDIR/app.conf"
}

keystore() { # the signing key: made once, needed for every update of the app
	local ks="$APPDIR/release.jks" pw="$APPDIR/keystore.env"
	if [ -f "$ks" ] && [ -f "$pw" ]; then return; fi
	[ -f "$ks" ] && die "$(t key_orphan "$ks" "$pw") (ANDROID_KEYSTORE_PASSWORD, ANDROID_KEY_ALIAS, ANDROID_KEY_PASSWORD)"
	local pass; pass=$(rand_alnum 32)
	say "$(t key_create)"
	docker run --rm -v "$APPDIR:/keys" --entrypoint keytool "$BUILDER" -genkeypair -v \
		-keystore /keys/release.jks -storetype PKCS12 -storepass "$pass" -keypass "$pass" \
		-alias release -keyalg RSA -keysize 4096 -validity 10000 \
		-dname "CN=${APP[APP_NAME]//,/ }, O=${APP[WEB_DOMAIN]}" >/dev/null 2>&1 || die "$(t key_fail)"
	docker run --rm -v "$APPDIR:/keys" --entrypoint chown "$BUILDER" "$(id -u):$(id -g)" /keys/release.jks
	printf 'ANDROID_KEYSTORE_PASSWORD=%s\nANDROID_KEY_ALIAS=release\nANDROID_KEY_PASSWORD=%s\n' "$pass" "$pass" >"$pw"
	chmod 600 "$ks" "$pw"
	NEW_KEY=y
}

build_app() {
	title "$(t build_title "${APP[APP_NAME]}" "${APP[APP_ID]}")"
	app_save
	source_tree
	BUILDER="librequenda-app-builder:${REF//\//-}"
	say "$(t builder_prep)"
	docker build -q -t "$BUILDER" "$SRC/app/builder" >/dev/null || die "$(t builder_fail)"
	NEW_KEY=n
	keystore

	local logo_mount=()
	[ -n "${APP[LOGO]}" ] && logo_mount=(-v "${APP[LOGO]}:/in/logo:ro")
	local version=${ENV[LIBREQUENDA_VERSION]:-latest}
	[ "$version" = latest ] && version="$REF-$(date +%Y%m%d)"
	say "$(t building)"
	docker run --rm \
		-v "$SRC:/src:ro" -v "$APPDIR/release.jks:/in/release.jks:ro" "${logo_mount[@]}" -v "$APPDIR:/out" \
		-v librequenda-gradle:/root/.gradle \
		--env-file "$APPDIR/keystore.env" \
		-e APP_ID="${APP[APP_ID]}" -e APP_NAME="${APP[APP_NAME]}" -e ICON_BACKGROUND="${APP[ICON_BACKGROUND]}" \
		-e API_URL="https://${APP[WEB_DOMAIN]}/api" \
		-e VERSION_NAME="$version" -e VERSION_CODE="$(($(date +%s) / 60))" \
		-e OUT_OWNER="$(id -u):$(id -g)" \
		"$BUILDER" || die "$(t build_fail)"
	ok "$(t built "$APPDIR/${APP[APP_ID]}.apk" "$APPDIR/${APP[APP_ID]}.aab")"

	say ""
	say "  ${bold}${APP[APP_ID]}.apk${off}  $(t apk_line)"
	say "  ${bold}${APP[APP_ID]}.aab${off}  $(t aab_line)"
	if [ "$NEW_KEY" = y ]; then
		say ""
		warn "$(t key_backup "$APPDIR/release.jks" "$APPDIR/keystore.env")"
	fi
	note "$(t app_update "cd $DIR && bash install.sh app")"
}

# the server must let sign-in return to the app: ANDROID_APP_IDS, then the setup runs again
register_app() {
	local id=${APP[APP_ID]}
	if [ ! -f "$DIR/.env" ] || [ ! -f "$DIR/docker-compose.yaml" ]; then
		say ""
		warn "$(t reg_missing "$DIR")"
		say "    ANDROID_APP_IDS=$id        ${dim}$(t reg_comma)${off}"
		say "    docker compose up -d --wait"
		return
	fi
	[[ ",${ENV[ANDROID_APP_IDS]:-}," == *",$id,"* ]] && return
	add_app_id "$id"
	cp "$DIR/.env" "$DIR/.env.backup-$(date +%Y%m%d-%H%M%S)"
	local example="$DIR/.env.example"; [ -f "$example" ] || fetch .env.example "$example"
	env_write "$example" "$DIR/.env"
	ok "$(t reg_added "$id" "$DIR/.env")"
	start_stack
}

check_registered() { # the server says which apps may sign in (config.json)
	local id=${APP[APP_ID]} json url="https://${APP[WEB_DOMAIN]}/config.json"
	json=$(curl -fsS --max-time 10 "$url" 2>/dev/null) || { warn "$(t chk_unreach "$url")"; return; }
	if grep -q "\"$id\"" <<<"$json"; then
		ok "$(t chk_ok "$id")"
	else
		warn "$(t chk_missing "$id")"
		if [ "${ENV[ZITADEL_SETUP_REVOKE_TOKEN]:-false}" = true ]; then warn "$(t chk_revoked)"
		else warn "$(t chk_logs)"; fi
	fi
}

install_app() {
	check_docker
	need curl "$(t why_curl)"
	screen_size
	mkdir -p "$DIR"
	[ -f "$DIR/.env" ] && env_load "$DIR/.env"
	while true; do
		app_questions
		ui_yesno "$(t app_confirm "${APP[APP_NAME]}" "${APP[APP_ID]}" "${APP[WEB_DOMAIN]}")" y && break
	done
	clear
	register_app
	build_app
	check_registered
}

# ============================================================
START_PWD=$PWD
DIR=$(realpath -m "${LIBREQUENDA_DIR:-$PWD}")
case "${1:-}" in
	-h|--help) awk 'NR > 2 && /^# =+$/ { exit } NR > 2' "$0" | sed 's/^# \{0,2\}//'; exit 0 ;;
esac
exec 3</dev/tty || die "run it in a terminal: it asks questions"
pick_language
case "${1:-}" in
	"") install_server ;;
	app) install_app ;;
	*) die "$(t unknown_cmd "$1")" ;;
esac
