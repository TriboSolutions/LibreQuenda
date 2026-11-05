# Check-in at the entrance

Councils need to know who actually turned up: it frees no-show turns, feeds the attendance indicators and keeps the facility for the people who booked it.

**What LibreQuenda does today.** The facility has a reader at the door (`/lector`): any phone, tablet or small computer with a camera, paired to the facility with a device key from `/<council>/admin`. The resident shows one of two QR codes:

- **Booking QR**: each booking has its own (in the app, in the confirmation and reminder emails, and printable by staff for someone without an account). The reader lets the person in from 15 minutes before the start until the end.
- **Flash booking QR**: the personal QR in the account menu. It changes every minute and works once, so a screenshot is useless. If the turn under way (or the one about to start) is free, scanning it books it on the spot, within the usual limits and rules.

Staff can also book from the desk ("Reserva manual"), print the entry QR for the person, and cancel.

The resident can also check in from their own phone, reading the facility instead of being read. Both ways are switched on per facility in the facility editor's **Entrada** tab, and both need the facility's map pin:

- **Printed QR at the door**: the tab prints a sheet with the facility's QR. It opens `/<council>/<facility>/entrada` with a random token, so the address alone does not work. "New QR" changes the token and the sheets already printed stop working.
- **Entry code**: the tab shows the facility's fixed 6-digit code and prints it for the door. It comes from an HMAC of a per-facility secret, so "New code" gives a new one and the printed copies stop working. The resident types it on the same `/entrada` page; their booking still has to be open for entry (from 15 minutes before it starts until it ends), and the endpoint has the strict rate limit, so guessing codes is slow.

Either way the phone's location is required: within the facility's distance (100 m unless the council changes it, between 25 and 1000 m) of its pin, with an accuracy of 150 m or better. The browser asks the resident; the API checks the location and drops it, it is never stored. A location can be faked by someone determined, so these two are the cheap options, not the strict ones. Then the check-in is the reader's own: the person's booking for this turn goes in. Without one, the page offers **Book and check in**, a flash booking of the free turn under the usual rules. The council can turn flash booking off for the facility in the same tab, which also turns it off for the personal QR at the readers.

**Staff confirmation** is the third way the tab switches on, and needs no pin. In the "Reserva manual" calendar, staff open a booking and press **Mark as in**, from 15 minutes before the start until the turn ends, as at a reader. It works for bookings by name too, and the booking shows who marked it. A staff entry can be undone for 5 minutes, in case they picked the wrong booking; entries from a reader or a phone cannot.

Every entry records how it happened (`reader`, `printed_qr`, `entry_code` or `staff`), and the council's data export includes it.

**Other ways it could work (not built yet).** The options below let a council choose by budget and by how much it cares about cheating. They would all end in the same check-in on the booking, so the indicators stay the same.

| Way | At the door | How the resident checks in | Cost | Hard to cheat? |
| --- | --- | --- | --- | --- |
| Booking or flash QR (built) | A device with a camera running `/lector` | Shows the QR from their phone or a printout | A cheap phone or tablet | Yes: tokens are per booking or change every minute |
| Printed QR (built) | A sheet of paper with the facility's QR | Scans it with their phone; the app checks in their booking for that facility and turn | Nearly free | Low: a photo of the sheet works anywhere, so the phone's location is always checked |
| Entry code (built) | A sheet at the door or the desk with the facility's fixed code | Types the code on their phone | Nearly free | Low on its own: the code does not change until the council makes a new one; the phone's location is always checked |
| NFC tag | A sticker on the door | Taps it with their phone, which opens the check-in page | A few euros per door | Low with a plain tag (it can be copied). High with a tag that signs every tap (e.g. NTAG 424 DNA "SUN" messages), which the API checks |
| Staff confirmation (built) | Nobody, or the desk | Staff mark them as in from the booking on the "Reserva manual" calendar | Staff time | As good as the staff |

What the ones not built yet would need:

- **NFC tag:** a plain tag can hold the `/entrada` address of the printed QR today. A signed one needs the tag's key, stored per door, and a check of the tap counter so a recorded tap cannot be replayed.
