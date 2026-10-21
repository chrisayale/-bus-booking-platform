import sys

file_path = r"internal/whatsapp/handler.go"

with open(file_path, 'r', encoding='utf-8') as f:
    content = f.read()

lines = content.splitlines(True)  # keep line endings

# Step 1: Remove the first occurrence of the case label and its block.
i = 0
while i < len(lines):
    if lines[i].strip() == "case StepTicketActionMenu:":
        # Found the first occurrence
        indent = lines[i][:len(lines[i]) - len(lines[i].lstrip())]
        j = i + 1
        while j < len(lines):
            line = lines[j]
            stripped = line.lstrip()
            if stripped.startswith("case ") or stripped.startswith("default:"):
                if line[:len(indent)] == indent:
                    break
            j += 1
        # Remove lines[i:j]
        del lines[i:j]
        break
    i += 1

# Step 2: In the modified lines, find the case label (now the second occurrence) and insert the variable declarations block after it and before the inner switch.
# We'll look for the case label again.
i = 0
while i < len(lines):
    if lines[i].strip() == "case StepTicketActionMenu:":
        indent = lines[i][:len(lines[i]) - len(lines[i].lstrip())]
        # Variable declarations block as a string (without indentation)
        variable_declarations_block = """isAuthorizedAgent := len(s.AllowedAgencyIDs) > 0
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
   Répondez avec le numéro de votre choix.`, s.CheckPhone)"""
        # Split the block into lines
        block_lines = variable_declarations_block.splitlines(True)
        # Add indent to each line
        block_lines = [indent + line if line.strip() != '' else line for line in block_lines]
        # Insert after the case label line
        lines = lines[:i+1] + block_lines + lines[i+1:]
        break
    i += 1

# Write the file back
with open(file_path, 'w', encoding='utf-8') as f:
    f.writelines(lines)
