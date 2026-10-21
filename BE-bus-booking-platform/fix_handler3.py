import sys

file_path = r"internal/whatsapp/handler.go"

with open(file_path, 'r', encoding='utf-8') as f:
    lines = f.readlines()

# Find the line index of the case label for StepTicketActionMenu
case_index = -1
for i, line in enumerate(lines):
    if line.strip() == "case StepTicketActionMenu:":
        case_index = i
        break

if case_index == -1:
    print("Error: case StepTicketActionMenu: not found")
    sys.exit(1)

# Find the end of the case block: the next line that starts with "case " or "default:" at the same indentation (which is 0)
# We'll look for the next line that, after stripping, starts with "case " or "default:" and has no leading whitespace.
end_index = len(lines)  # default to end of file
for i in range(case_index + 1, len(lines)):
    stripped = lines[i].lstrip()
    if stripped.startswith("case ") or stripped.startswith("default:"):
        # Check if the line has no leading whitespace (since our case label has no leading whitespace)
        if lines[i] == stripped:  # meaning no leading whitespace
            end_index = i
            break

# The block to replace is from case_index to end_index-1
# We want to replace it with the new block.

# Define the new block lines (without indentation, because the case label has no indentation)
new_block_lines = [
    "case StepTicketActionMenu:\n",
    "	isAuthorizedAgent := len(s.AllowedAgencyIDs) > 0\n",
    "	authorized := false\n",
    "	if isAuthorizedAgent {\n",
    "	    for _, allowedID := range s.AllowedAgencyIDs {\n",
    "	        if allowedID == s.TicketOrganizationID {\n",
    "		    authorized = true\n",
    "		    break\n",
    "	        }\n",
    "	    }\n",
    "	}\n",
    "	fullMenu := fmt.Sprintf(`Pour le billet %s :\n   1️⃣ Voir l'état du billet\n   2️⃣ Valider le paiement cash\n   3️⃣ Check-in du billet (Embarquement)\n\n   Répondez avec le numéro de votre choix.`, s.CheckPhone)\n",
    "	restrictedMenu := fmt.Sprintf(`Pour le billet %s :\n   1️⃣ Voir l'état du billet\n   Répondez avec le numéro de votre choix.`, s.CheckPhone)\n",
    "	switch text {\n",
    "	case \"1\":\n",
    "	    // View ticket status (allowed for all)\n",
    "	    ticket, err := GetTicketByDownloadToken(s.CheckPhone)\n",
    "	    if err != nil {\n",
    "		log.Printf(\"Error retrieving ticket: %v\", err)\n",
    "		resetSession(s.UserPhone)\n",
    "		return \"Erreur lors de la récupération du billet. Tapez *menu* pour réessayer.\"\n",
    "	    }\n",
    "	    if ticket == nil {\n",
    "		resetSession(s.UserPhone)\n",
    "		return \"Aucun billet trouvé pour ce code. Tapez *menu* pour réessayer.\"\n",
    "	    }\n",
    "	    // Format ticket details\n",
    "	    var b strings.Builder\n",
    "	    b.WriteString(\"Voici les détails du billet :\n\n\")\n",
    "	    b.WriteString(fmt.Sprintf(\"   Passenger : %s\n\", ticket.PassengerName))\n",
    "	    b.WriteString(fmt.Sprintf(\"   Siège     : %s\n\", ticket.SeatNumber))\n",
    "	    b.WriteString(fmt.Sprintf(\"   Prix      : %.2f USD\n\", ticket.Price))\n",
    "	    b.WriteString(fmt.Sprintf(\"   Télécharger : %s\n\n\", ticket.DownloadURL))\n",
    "	    return b.String()\n",
    "	case \"2\":\n",
    "	    if !authorized {\n",
    "		return \"Accès non autorisé. Tapez *menu* pour revenir au menu principal.\"\n",
    "	    }\n",
    "	    // Validate cash payment\n",
    "	    err := UpdateTicketPaymentStatus(s.CheckPhone)\n",
    "	    if err != nil {\n",
    "		log.Printf(\"Error updating payment status: %v\", err)\n",
    "		resetSession(s.UserPhone)\n",
    "		return \"Erreur lors de la validation du paiement. Tapez *menu* pour réessayer.\"\n",
    "	    }\n",
    "	    resetSession(s.UserPhone)\n",
    "	    return \"Paiement validé avec succès. Tapez *menu* pour revenir au menu principal.\"\n",
    "	case \"3\":\n",
    "	    if !authorized {\n",
    "		return \"Accès non autorisé. Tapez *menu* pour revenir au menu principal.\"\n",
    "	    }\n",
    "	    // Check-in the ticket\n",
    "	    err := UpdateTicketCheckinStatus(s.CheckPhone)\n",
    "	    if err != nil {\n",
    "		log.Printf(\"Error updating checkin status: %v\", err)\n",
    "		resetSession(s.UserPhone)\n",
    "		return \"Erreur lors du check-in du billet. Tapez *menu* pour réessayer.\"\n",
    "	    }\n",
    "	    resetSession(s.UserPhone)\n",
    "	    return \"Billet enregistré comme embarqué. Tapez *menu* pour revenir au menu principal.\"\n",
    "	default:\n",
    "	    if authorized {\n",
    "		return fullMenu\n",
    "	    } else {\n",
    "		return restrictedMenu\n",
    "	    }\n",
    "	}\n",
]

# Replace the lines
lines = lines[:case_index] + new_block_lines + lines[end_index:]

# Write the file back
with open(file_path, 'w', encoding='utf-8') as f:
    f.writelines(lines)
