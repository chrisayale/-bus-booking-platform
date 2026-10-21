import sys

file_path = r"internal/whatsapp/handler.go"

with open(file_path, 'r', encoding='utf-8') as f:
    lines = f.readlines()

case_index = -1
for i, line in enumerate(lines):
    if line.strip() == "case StepTicketActionMenu:":
        case_index = i
        break

if case_index == -1:
    print("Error: case StepTicketActionMenu: not found")
    sys.exit(1)

indent = lines[case_index][:len(lines[case_index]) - len(lines[case_index].lstrip())]

block_lines = [
    "isAuthorizedAgent := len(s.AllowedAgencyIDs) > 0",
    "authorized := false",
    "if isAuthorizedAgent {",
    "    for _, allowedID := range s.AllowedAgencyIDs {",
    "        if allowedID == s.TicketOrganizationID {",
    "            authorized = true",
    "            break",
    "        }",
    "    }",
    "}",
    "fullMenu := fmt.Sprintf(`Pour le billet %s :\n   1️⃣ Voir l'état du billet\n   2️⃣ Valider le paiement cash\n   3️⃣ Check-in du billet (Embarquement)\n\n   Répondez avec le numéro de votre choix.`, s.CheckPhone)",
    "restrictedMenu := fmt.Sprintf(`Pour le billet %s :\n   1️⃣ Voir l'état du billet\n   Répondez avec le numéro de votre choix.`, s.CheckPhone)"
]

block_lines = [indent + line for line in block_lines]

lines = lines[:case_index+1] + block_lines + lines[case_index+1:]

with open(file_path, 'w', encoding='utf-8') as f:
    f.writelines(lines)
