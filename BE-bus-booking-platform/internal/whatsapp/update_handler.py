import re

# Read the file with UTF-8 encoding
with open('handler.go', 'r', encoding='utf-8') as f:
    content = f.read()

# 1. Update welcomeMenu to have 5 options
new_welcome = """const welcomeMenu = `Bienvenue sur GUV (Guichet unique des voyages) 🚌

1️⃣ Recherche directe de voyages
2️⃣ Recherche par agence
3️⃣ Créer une agence
4️⃣ 🎫 Vérifier mon billet (Check Ticket)
5️⃣ Aide & Assistance

Répondez avec le numéro de votre choix.`"""
# Replace the old welcomeMenu block
pattern = r'const welcomeMenu = `.*?\n\s*`\n'
content = re.sub(pattern, new_welcome + '\n\n', content, flags=re.DOTALL)

# 2. Update the StepMenu case in the process function
# Find the start of the process function
process_start = content.find('func process')
if process_start == -1:
    print("Error: func process not found")
    exit(1)

# Find the outer switch statement
switch_start = content.find('switch s.Step {', process_start)
if switch_start == -1:
    print("Error: switch s.Step not found in process function")
    exit(1)

# Find the case StepMenu: after switch_start
case_step_menu_start = content.find('case StepMenu:', switch_start)
if case_step_menu_start == -1:
    print("Error: case StepMenu: not found in process function")
    exit(1)

# Split into lines to find the end of the case block
lines = content.split('\n')
line_index = -1
for i, line in enumerate(lines):
    if line.startswith('case StepMenu:'):
        line_index = i
        break
if line_index == -1:
    print("Error: case StepMenu: not found in lines")
    exit(1)

# Find the next line that starts a new case or default at the same level (no leading whitespace)
i = line_index + 1
while i < len(lines):
    stripped = lines[i].lstrip()
    if stripped.startswith('case ') or stripped.startswith('default:'):
        if lines[i].startswith(stripped):
            break
    i += 1

# Build the new StepMenu case block
new_step_menu_block = """case StepMenu:
	switch text {
	case "1":
		s.PreviousStep = s.Step
		s.Step = StepSearchOrigin
		return getPromptForStep(s)
	case "2":
		agencies, err := getActiveAgencies()
		if err != nil {
			return "Erreur lors de la récupération des agences. Tapez *menu* pour réessayer."
		}
		log.Printf("Found %d active agencies for phone %s", len(agencies), phone)
		if len(agencies) == 0 {
			return "Aucune agence active trouvée. Tapez *menu* pour réessayer."
		}
		s.Agencies = agencies
		s.PreviousStep = s.Step
		s.Step = StepAgencyChoice
		return getPromptForStep(s)
	case "3":
		s.PreviousStep = s.Step
		s.Step = StepAgencyName
		return getPromptForStep(s)
	case "4":
		s.PreviousStep = s.Step
		s.Step = StepCheckMenu
		return getPromptForStep(s)
	case "5":
		// Aide & Assistance
		return "📞 Pour toute assistance ou information, notre service client est à votre disposition :\n- Ligne directe & WhatsApp : +243974645174\n- Email : ayaletrans@gmail.com"
	default:
		return welcomeMenu
}"""

# Replace the lines
lines[line_index:i] = [new_step_menu_block]
content = '\n'.join(lines)

# 3. Add a case for StepCheckMenu in the process function
# Find the line index of the case StepMenu: we just inserted
lines = content.split('\n')
line_index = -1
for i, line in enumerate(lines):
    if line.startswith('case StepMenu:'):
        line_index = i
        break
if line_index == -1:
    print("Error: case StepMenu: not found after update")
    exit(1)

# Find the end of the StepMenu case block we just inserted
i = line_index + 1
while i < len(lines):
    stripped = lines[i].lstrip()
    if stripped.startswith('case ') or stripped.startswith('default:'):
        if lines[i].startswith(stripped):
            break
    i += 1

# Build the StepCheckMenu case block
new_step_check_menu_block = """case StepCheckMenu:
	switch text {
	case "1":
		s.PreviousStep = s.Step
		s.Step = StepPassengerCheckType
		return getPromptForStep(s)
	case "2":
		s.PreviousStep = s.Step
		s.Step = StepAgentPinInput
		return getPromptForStep(s)
	default:
		return getPromptForStep(s)
}"""
# Insert the block at position i
lines.insert(i, new_step_check_menu_block)
content = '\n'.join(lines)

# Write the file back with UTF-8 encoding
with open('handler.go', 'w', encoding='utf-8') as f:
    f.write(content)

print("handler.go updated successfully")
