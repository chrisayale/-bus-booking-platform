import sys

file_path = r"internal/whatsapp/handler.go"

with open(file_path, 'r', encoding='utf-8') as f:
    lines = f.readlines()

# Step 1: Remove the first occurrence of "case StepTicketActionMenu:" and its block.
# We'll find the first occurrence.
first_occurrence_index = -1
for i, line in enumerate(lines):
    if line.strip() == "case StepTicketActionMenu:":
        first_occurrence_index = i
        break

if first_occurrence_index == -1:
    print("Error: first occurrence of case StepTicketActionMenu: not found")
    sys.exit(1)

# Find the end of the first occurrence block: the next line that starts with "case " or "default:" at the same indentation.
indent_first = lines[first_occurrence_index][:len(lines[first_occurrence_index]) - len(lines[first_occurrence_index].lstrip())]
j = first_occurrence_index + 1
while j < len(lines):
    line = lines[j]
    stripped = line.lstrip()
    if stripped.startswith("case ") or stripped.startswith("default:"):
        if line[:len(indent_first)] == indent_first:
            break
    j += 1
# Remove lines[first_occurrence_index:j]
del lines[first_occurrence_index:j]

# Step 2: Now, find the second occurrence of "case StepTicketActionMenu:" in the modified lines.
# We'll search for it.
second_occurrence_index = -1
for i, line in enumerate(lines):
    if line.strip() == "case StepTicketActionMenu:":
        second_occurrence_index = i
        break

if second_occurrence_index == -1:
    print("Error: second occurrence of case StepTicketActionMenu: not found")
    sys.exit(1)

# We will keep the case label line and replace the block after it until the next case or default at the same indentation.
indent_second = lines[second_occurrence_index][:len(lines[second_occurrence_index]) - len(lines[second_occurrence_index].lstrip())]
# Find the end of the second occurrence block: the next line that starts with "case " or "default:" at the same indentation.
k = second_occurrence_index + 1
while k < len(lines):
    line = lines[k]
    stripped = line.lstrip()
    if stripped.startswith("case ") or stripped.startswith("default:"):
        if line[:len(indent_second)] == indent_second:
            break
    k += 1
# The block to replace is from second_occurrence_index+1 to k-1.
# We will replace it with the new block.

# Define the new block as a string (without indentation)
new_block = """isAuthorizedAgent := len(s.AllowedAgencyIDs) > 0
authorized := false
if isAuthorizedAgent {
    for _, allowedID := range s.AllowedAgencyIDs {
        if allowedID == s.TicketOrganizationID {
            authorized = true
            break
        }
    }
}
fullMenu := fmt.Sprintf(`Pour le billet %s :
   1️⃣ Voir l'état du billet
   2️⃣ Valider le paiement cash
   3️⃣ Check-in du billet (Embarquement)

   Répondez avec le numéro de votre choix.`, s.CheckPhone)
restrictedMenu := fmt.Sprintf(`Pour le billet %s :
   1️⃣ Voir l'état du billet
   Répondez avec le numéro de votre choix.`, s.CheckPhone)
switch text {
case \"1\":
    // View ticket status (allowed for all)
    ticket, err := GetTicketByDownloadToken(s.CheckPhone)
    if err != nil {
        log.Printf(\"Error retrieving ticket: %v\", err)
        resetSession(s.UserPhone)
        return \"Erreur lors de la récupération du billet. Tapez *menu* pour réessayer.\"
    }
    if ticket == nil {
        resetSession(s.UserPhone)
        return \"Aucun billet trouvé pour ce code. Tapez *menu* pour réessayer.\"
    }
    // Format ticket details
    var b strings.Builder
    b.WriteString(\"Voici les détails du billet :\n\n\")
    b.WriteString(fmt.Sprintf(\"   Passenger : %s\n\", ticket.PassengerName))
    b.WriteString(fmt.Sprintf(\"   Siège     : %s\n\", ticket.SeatNumber))
    b.WriteString(fmt.Sprintf(\"   Prix      : %.2f USD\n\", ticket.Price))
    b.WriteString(fmt.Sprintf(\"   Télécharger : %s\n\n\", ticket.DownloadURL))
    return b.String()
case \"2\":
    if !authorized {
        return \"Accès non autorisé. Tapez *menu* pour revenir au menu principal.\"
    }
    // Validate cash payment
    err := UpdateTicketPaymentStatus(s.CheckPhone)
    if err != nil {
        log.Printf(\"Error updating payment status: %v\", err)
        resetSession(s.UserPhone)
        return \"Erreur lors de la validation du paiement. Tapez *menu* pour réessayer.\"
    }
    resetSession(s.UserPhone)
    return \"Paiement validé avec succès. Tapez *menu* pour revenir au menu principal.\"
case \"3\":
    if !authorized {
        return \"Accès non autorisé. Tapez *menu* pour revenir au menu principal.\"
    }
    // Check-in the ticket
    err := UpdateTicketCheckinStatus(s.CheckPhone)
    if err != nil {
        log.Printf(\"Error updating checkin status: %v\", err)
        resetSession(s.UserPhone)
        return \"Erreur lors du check-in du billet. Tapez *menu* pour réessayer.\"
    }
    resetSession(s.UserPhone)
    return \"Billet enregistré comme embarqué. Tapez *menu* pour revenir au menu principal.\"
default:
    if authorized {
        return fullMenu
    } else {
        return restrictedMenu
    }
}
"""

# Split the new block into lines and ensure each line ends with a newline.
# We split by newline and then add a newline if the line doesn't already end with one.
# But note: the string may have a trailing newline? We'll use splitlines and then add newline.
new_block_lines = new_block.splitlines(True)
# If the last line does not end with a newline, we add it? But splitlines(True) keeps the newline.
# Now, add indentation to each line.
new_block_lines = [indent_second + line if line.strip() != '' else line for line in new_block_lines]

# Replace the lines from second_occurrence_index+1 to k-1 with the new block.
lines = lines[:second_occurrence_index+1] + new_block_lines + lines[k:]

# Write the file back
with open(file_path, 'w', encoding='utf-8') as f:
    f.writelines(lines)
