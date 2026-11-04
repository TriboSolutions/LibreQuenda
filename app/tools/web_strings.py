"""Generates app/app/src/main/res/values*/strings_web.xml from the web's translations, so the
app says exactly what the web does. Run from the repository root after changing web/src/locales:

    python3 app/tools/web_strings.py web/src app/app/src/main/res
"""
import json, re, sys, os
web, res = sys.argv[1], sys.argv[2]
P = 'profile'; C = 'common'; B = 'bookings'; I = 'invites'; H = 'home'; K = 'councils'; N = 'notifications'; F = 'facility'; E = 'entrance'; L = 'legal'
KEYS = {
 'profile_title': (P,'title'), 'profile_signed_out': (P,'signIn'), 'profile_subtitle': (P,'subtitle'),
 'profile_incomplete': (P,'incompleteBody'),
 'profile_details': (P,'details'), 'profile_photo': (P,'photo'),
 'birth_date': (P,'birthDate'), 'birth_date_help': (P,'birthDateHelp'), 'birth_date_placeholder': (P,'birthDatePlaceholder'),
 'birth_date_required': (P,'birthDateRequired'), 'phone': (P,'phone'), 'phone_help': (P,'phoneHelp'), 'optional': (P,'optional'),
 'save_profile': (P,'save'), 'saving': (P,'saving'), 'profile_saved': (P,'saved'),
 'verified': (P,'verified'), 'not_verified': (P,'notVerified'),
 'preferences': (P,'preferences'), 'preferences_help': (P,'preferencesHelp'), 'language_emails': (P,'language'),
 'emails': (P,'emails'), 'emails_help': (P,'emailsHelp'), 'reminders': (P,'reminders'),
 'recommendations': (P,'recommendations'), 'recommendations_help': (P,'recommendationsHelp'),
 'save_preferences': (P,'savePreferences'), 'preferences_saved': (P,'preferencesSaved'),
 'account_title': (P,'accountSettings.title'), 'account_help': (P,'accountSettings.help'), 'account_name': (P,'accountSettings.name'),
 'given_name': (P,'accountSettings.givenName'), 'family_name': (P,'accountSettings.familyName'), 'save_name': (P,'accountSettings.saveName'),
 'name_saved': (P,'accountSettings.nameSaved'), 'account_email': (P,'accountSettings.email'), 'change_email': (P,'accountSettings.changeEmail'),
 'new_email': (P,'accountSettings.newEmail'), 'new_email_help': (P,'accountSettings.newEmailHelp'), 'send_code': (P,'accountSettings.sendCode'),
 'code': (P,'accountSettings.code'), 'code_help': (P,'accountSettings.codeHelp'), 'verify': (P,'accountSettings.verify'),
 'resend': (P,'accountSettings.resend'), 'code_sent': (P,'accountSettings.codeSent'), 'email_verified': (P,'accountSettings.emailVerified'),
 'password': (P,'accountSettings.password'), 'current_password': (P,'accountSettings.currentPassword'), 'new_password': (P,'accountSettings.newPassword'),
 'repeat_password': (P,'accountSettings.repeatPassword'), 'mismatch': (P,'accountSettings.mismatch'), 'save_password': (P,'accountSettings.savePassword'),
 'password_saved': (P,'accountSettings.passwordSaved'), 'cancel': (P,'accountSettings.cancel'),
 'rule_uppercase': (P,'accountSettings.rule.uppercase'), 'rule_lowercase': (P,'accountSettings.rule.lowercase'),
 'rule_number': (P,'accountSettings.rule.number'), 'rule_symbol': (P,'accountSettings.rule.symbol'),
 'calendar_title': (P,'calendar.title'), 'calendar_help': (P,'calendar.help'), 'calendar_how': (P,'calendar.how'),
 'calendar_turn_on': (P,'calendar.turnOn'), 'calendar_address': (P,'calendar.address'), 'calendar_copy': (P,'calendar.copy'),
 'calendar_copied': (P,'calendar.copied'), 'calendar_private': (P,'calendar.private'), 'calendar_google': (P,'calendar.google'),
 'calendar_outlook': (P,'calendar.outlook'), 'calendar_outlook_work': (P,'calendar.outlookWork'), 'calendar_apple': (P,'calendar.apple'),
 'calendar_google_slow': (P,'calendar.googleSlow'), 'calendar_reset': (P,'calendar.reset'), 'calendar_reset_hint': (P,'calendar.resetHint'),
 'calendar_turn_off': (P,'calendar.turnOff'),
 'privacy_title': (P,'privacy.title'), 'privacy_help': (P,'privacy.help'), 'export_title': (P,'privacy.exportTitle'),
 'export_help': (P,'privacy.exportHelp'), 'export_json': (P,'privacy.exportJson'), 'export_csv': (P,'privacy.exportCsv'),
 'preparing': (P,'privacy.preparing'), 'delete_title': (P,'privacy.deleteTitle'), 'delete_help': (P,'privacy.deleteHelp'),
 'delete_account': (P,'privacy.delete'), 'delete_dialog_title': (P,'privacy.deleteDialogTitle'), 'delete_dialog_body': (P,'privacy.deleteDialogBody'),
 'delete_upcoming': (P,'privacy.deleteUpcoming'), 'delete_past': (P,'privacy.deletePast'), 'delete_sign_in': (P,'privacy.deleteSignIn'),
 'delete_confirm_label': (P,'privacy.deleteConfirmLabel'), 'delete_confirm': (P,'privacy.deleteConfirm'), 'deleting': (P,'privacy.deleting'),
 'deleted_title': (P,'privacy.deletedTitle'), 'deleted_body': (P,'privacy.deletedBody'),
 'retry': (C,'retry'), 'rate_limited': (C,'rateLimited'),
 'image_upload': (C,'imagePicker.upload'), 'image_change': (C,'imagePicker.change'), 'image_remove': (C,'imagePicker.remove'),
 'image_uploading': (C,'imagePicker.uploading'), 'image_hint': (C,'imagePicker.hint'), 'image_too_large': (C,'imagePicker.tooLarge'),
 'image_wrong_type': (C,'imagePicker.wrongType'),
 # my bookings
 'bookings_title': (I,'myBookings.title'), 'bookings_subtitle': (I,'myBookings.subtitle'),
 'bookings_history': (I,'myBookings.history'), 'bookings_empty_upcoming': (I,'myBookings.emptyUpcoming'),
 'bookings_empty_history': (I,'myBookings.emptyHistory'), 'bookings_signed_out': (I,'myBookings.signIn'), 'bookings_book_now': (I,'myBookings.bookNow'),
 'bookings_load_error': (B,'loadError'),
 'cancel_title': (B,'cancelDialog.title'), 'cancel_description': (B,'cancelDialog.description'), 'cancel_keep': (B,'cancelDialog.keep'),
 'cancel_confirm': (B,'cancelDialog.confirm'), 'cancel_confirming': (B,'cancelDialog.confirming'),
 'notice_cancelled': (B,'notice.cancelled'),
 # home reminders, as the web's (ProfileReminder)
 'incomplete_title': (P,'incompleteTitle'), 'incomplete_body': (P,'incompleteBody'), 'incomplete_action': (P,'incompleteAction'),
 'unverified_title': (P,'unverifiedTitle'), 'unverified_body': (P,'unverifiedBody'), 'unverified_action': (P,'unverifiedAction'),
 # facility search
 'search_placeholder': (K,'search'), 'search_no_results': (K,'noResults'), 'search_load_error': (K,'loadError'),
 'join': (H,'forYou.join'), 'joining': (H,'forYou.joining'), 'joined': (H,'forYou.joined'),
 'all_councils': (K,'allCouncils'),
 # reporting a problem at the facility, during the booking (the web's ReportIssueDialog)
 'issue_report': (F,'issue.report'), 'issue_intro': (F,'issue.description'), 'issue_category': (F,'issue.category'),
 'issue_choose': (F,'issue.chooseCategory'), 'issue_what': (F,'issue.what'), 'issue_what_help': (F,'issue.whatHelp'),
 'issue_photo': (F,'issue.photo'), 'issue_urgent': (F,'issue.urgent'), 'issue_urgent_help': (F,'issue.urgentHelp'),
 'issue_share': (F,'issue.shareContact'), 'issue_location_note': (F,'issue.locationNote'), 'issue_send': (F,'issue.send'), 'issue_sending': (F,'issue.sending'), 'issue_sent': (F,'issue.sent'),
 'issue_cat_equipment': (F,'issue.categories.equipment'), 'issue_cat_lighting': (F,'issue.categories.lighting'),
 'issue_cat_cleaning': (F,'issue.categories.cleaning'), 'issue_cat_changing_rooms': (F,'issue.categories.changing_rooms'),
 'issue_cat_accessibility': (F,'issue.categories.accessibility'), 'issue_cat_safety': (F,'issue.categories.safety'), 'issue_cat_other': (F,'issue.categories.other'),
 # changing a booking to another turn (the web's ChangeBookingDialog)
 'change': (B,'change'), 'change_intro': (B,'changeDialog.description'), 'change_when': (B,'changeDialog.when'),
 'change_current': (B,'changeDialog.current'), 'change_closed_day': (B,'changeDialog.closedDay'), 'change_pick': (B,'changeDialog.pick'),
 'change_confirm': (B,'changeDialog.confirm'), 'change_saving': (B,'changeDialog.saving'), 'change_error': (B,'changeDialog.error'),
 # legal information and support (the texts themselves go to assets/legal, below)
 'legal_group': (L,'group'), 'legal_updated': (L,'updated'), 'terms_title': (L,'terms.title'), 'terms_sub': (L,'terms.sub'),
 'privacy_title': (L,'privacy.title'), 'privacy_sub': (L,'privacy.sub'), 'support_group': (L,'support.group'),
 'support_title': (L,'support.title'), 'support_sub': (L,'support.sub'), 'support_help': (L,'support.help'),
 'support_email': (L,'support.email'), 'support_phone': (L,'support.phone'),
 # flash booking, every way in (the web's FlashBookingDialog)
 'flash_help': (P,'flash.help'), 'flash_ways': (P,'flash.ways'),
 'flash_reader_help': (P,'flash.reader.help'), 'flash_code_help': (P,'flash.code.help'), 'flash_locate': (P,'flash.code.locate'),
 'flash_locating': (P,'flash.code.locating'), 'flash_again': (P,'flash.code.again'), 'flash_code_none': (P,'flash.code.none'),
 'flash_facility': (P,'flash.code.facility'), 'flash_door_help': (P,'flash.door.help'), 'flash_nfc_help': (P,'flash.nfc.help'),
 'flash_staff_help': (P,'flash.staff.help'), 'flash_staff_locate': (P,'flash.staff.locate'), 'flash_staff_none': (P,'flash.staff.none'),
 'qr_private': (P,'personalQr.private'), 'qr_brightness': (P,'personalQr.brightness'), 'qr_failed': (P,'personalQr.failed'),
 'qr_incomplete': (P,'personalQr.incomplete'), 'qr_refresh_in': (P,'personalQr.refreshIn'), 'way_nfc': (B,'checkin.nfc.tab'), 'way_nfc_help': (B,'checkin.nfc.help'),
 'entrance_booked': (E,'result.booked'), 'entrance_not_booked': (E,'result.not_booked'),
 # checking in: the ways a facility takes, and what the entrance answers
 'checkin_button': (B,'checkin.button'), 'checkin_title': (B,'checkin.title'), 'checkin_done_at': (B,'checkin.done'), 'checkin_ways': (B,'checkin.ways'),
 'way_reader': (B,'checkin.reader.tab'), 'way_reader_help': (B,'checkin.reader.help'),
 'way_entry_code': (B,'checkin.entry_code.tab'), 'way_entry_code_help': (B,'checkin.entry_code.help'),
 'way_printed_qr': (B,'checkin.printed_qr.tab'), 'way_printed_qr_help': (B,'checkin.printed_qr.help'),
 'way_staff': (B,'checkin.staff.tab'), 'way_staff_help': (B,'checkin.staff.help'), 'way_staff_help_contacts': (B,'checkin.staff.helpWithContacts'),

 'entry_code': (E,'code'), 'entry_code_help': (E,'codeHelp'), 'checking': (E,'checking'), 'locating': (K,'locating'),
 'inaccurate_help': (E,'inaccurateHelp'), 'scan_qr': (E,'scanQr'),
 **{f'entrance_{r}': (E, f'result.{r}') for r in ['checked_in', 'already_checked_in', 'too_early', 'ended', 'cancelled', 'method_off',
    'invalid_qr', 'wrong_code', 'no_facility_location', 'location_required', 'location_inaccurate', 'too_far', 'no_booking', 'refused']},
 # a facility's page and the booking's confirmation
 'minutes': (H,'availability.minutes'), 'book_title': (F,'bookTitle'), 'no_free': (F,'noFree'), 'book_at': (H,'availability.bookAt'),
 'confirm_title': (B,'dialog.title'), 'duration': (B,'dialog.duration'),
 'cancellation': (B,'dialog.cancellation'), 'cancellation_until': (B,'dialog.cancellationUntil'), 'rules_label': (B,'dialog.rules'),
 'confirm_booking': (B,'dialog.confirm'), 'confirming': (B,'dialog.confirming'),
 'accept_rules': (B,'dialog.acceptRules'), 'make_public': (B,'dialog.public'), 'make_public_help': (B,'dialog.publicHelp'),
 'public_description': (B,'public.description'), 'public_description_help': (B,'public.descriptionHelp'),
 # a booking's details and actions, as /reservas
 'invite': (I,'actions.invite'), 'invite_share_text': (I,'dialog.shareText'),
 'leave_title': (I,'leaveDialog.title'), 'leave_description': (I,'leaveDialog.description'), 'leave_keep': (I,'leaveDialog.keep'),
 'leave_confirm': (I,'leaveDialog.confirm'), 'leave_confirming': (I,'leaveDialog.confirming'), 'notice_left': (I,'notice.left'),
 'role_invited_by': (I,'role.invitedBy'), 'role_invited': (I,'role.invited'),
 # the council's events, joined like a public booking
 'role_event': (I,'role.event'), 'event_label': (F,'events.label'), 'event_no_description': (B,'public.noEventDescription'),
 'public_label': (B,'public.label'), 'public_turn_title': (B,'public.turnTitle'), 'public_edit': (B,'public.edit'), 'public_save': (B,'public.save'),
 'public_confirm_title': (B,'public.confirmTitle'), 'public_confirm_text': (B,'public.confirmDescription'), 'public_confirm': (B,'public.confirm'),
 'public_publishing': (B,'public.publishing'), 'public_keep_private': (B,'public.keepPrivate'),
 'public_no_description': (B,'public.noDescription'), 'public_person': (B,'public.person'), 'public_full': (B,'public.full'), 'public_started': (B,'public.started'), 'public_toggle_help': (B,'public.help'), 'people_count': (B,'public.people'),
 'checkin_from': (B,'checkin.from'),
 'directions': (F,'directions'), 'conditions': (F,'conditions'), 'min_age': (F,'minAge'),
 'usage_rules': (F,'rules'), 'no_rules': (F,'noRules'), 'no_location': (F,'noLocation'),
 'accessibility': (I,'details.accessibility'), 'cancel_until': (I,'details.cancelUntil'),
 # the notices as Android notifications: same words as the web's list
 **{f'notif_{t}': (N, f'type.{t}') for t in ['booking_confirmed', 'booking_reminder', 'booking_changed', 'booking_cancelled',
    'booking_cancelled_by_council', 'booking_cancelled_by_closure', 'booking_cancelled_by_staff', 'council_admin_granted',
    'issue_solved', 'issue_rejected', 'booking_joined', 'issue_reported']}, 'council': (K,'council'), 'slots_full': (H,'availability.full'),
 'status_cancelled': (B,'status.cancelled'), 'status_attended': (B,'status.attended'), 'status_no_show': (B,'status.noShow'),
 'status_ended': (B,'status.ended'), 'status_checked_in': (B,'status.checkedIn'),
}
PLURALS = {'rule_length': (P,'accountSettings.rule.length'), 'results_count': (K,'showing'), 'public_spots': (H,'forYou.spots'), 'rule_slot': (F,'ruleSlot'), 'rule_cancel': (F,'ruleCancel')}
def get(d, path):
    for part in path.split('.'): d = d[part]
    return d
# the notices take their values in a fixed order, whatever order each language puts them in
FIXED = ['facility', 'day', 'time', 'council', 'name', 'from', 'until']

def conv(text, key):
    if key == 'image_hint': text = re.split(r'(?<=\.)\s', text)[0]  # the drag-and-drop half is web only
    n = 0; seen = {f: i + 1 for i, f in enumerate(FIXED)} if key.startswith('notif_') else {}
    def ph(m):
        nonlocal n
        name = m.group(1)
        if name not in seen: n += 1; seen[name] = n
        return f'%{seen[name]}$' + ('d' if name == 'count' else 's')
    text = re.sub(r'\{\{(\w+)\}\}', ph, text)
    text = text.replace('&', '&amp;').replace('<', '&lt;').replace("'", "\\'").replace('"', '\\"')
    if text.startswith('@') or text.startswith('?'): text = '\\' + text
    return text
for lang, folder in [('gl','values'),('es','values-es'),('ca','values-ca'),('en','values-en')]:
    data = {ns: json.load(open(f'{web}/locales/{lang}/{ns}.json')) for ns in (P, C, B, I, H, K, N, F, E, L)}
    out = ['<?xml version="1.0" encoding="utf-8"?>',
           '<!-- Generated by app/tools/web_strings.py from web/src/locales/%s: do not edit by hand. -->' % lang,
           '<resources>']
    for key, (ns, path) in KEYS.items():
        out.append(f'    <string name="{key}">{conv(get(data[ns], path), key)}</string>')
    for key, (ns, path) in PLURALS.items():
        d = data[ns]
        for part in path.split('.')[:-1]: d = d[part]
        last = path.split('.')[-1]
        items = [('one', d[f'{last}_one']), ('other', d[f'{last}_other'])]
        if lang in ('es', 'ca'): items.insert(1, ('many', d[f'{last}_other']))
        out.append(f'    <plurals name="{key}">')
        for q, t in items: out.append(f'        <item quantity="{q}">{conv(t, key)}</item>')
        out.append('    </plurals>')
    out.append('</resources>\n')
    open(f'{res}/{folder}/strings_web.xml', 'w').write('\n'.join(out))
# the terms and the privacy policy, whole, as the app shows them (ui/legal): the same JSON as the web
import os, shutil
os.makedirs(f'{res}/../assets/legal', exist_ok=True)
for lang in ('gl', 'es', 'ca', 'en'):
    shutil.copyfile(f'{web}/locales/{lang}/legal.json', f'{res}/../assets/legal/{lang}.json')
print('ok', len(KEYS) + len(PLURALS))
