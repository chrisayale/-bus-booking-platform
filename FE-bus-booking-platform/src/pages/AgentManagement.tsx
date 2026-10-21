import { useEffect, useState } from "react"
import { api } from "../api/client"
import { Trash2, Edit2, Plus, RefreshCw } from "lucide-react"
import type { User, Organization } from "../types"

export default function AgentManagement() {
  const [agents, setAgents] = useState<User[]>([])
  const [organizations, setOrganizations] = useState<Organization[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)
  const [selectedAgent, setSelectedAgent] = useState<User | null>(null)
  const [formLoading, setFormLoading] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)
  const [formSuccess, setFormSuccess] = useState<string | null>(null)

  // Form fields
  const [formData, setFormData] = useState({
    first_name: "",
    last_name: "",
    phone: "",
    pin: "",
    role: "agent" as "agent" | "controller",
    organizationIds: [] as string[],
    is_active: true,
  });

  const [searchTerm, setSearchTerm] = useState('');
  const [selectedOrgId, setSelectedOrgId] = useState<string | null>(null);
  const [selectedStatus, setSelectedStatus] = useState<'all' | 'active' | 'inactive'>('all');

  useEffect(() => {
    fetchAll();
  }, []);

  const fetchAll = async () => {
    try {
      setLoading(true);
      setError(null);
      const [agentsData, orgsData] = await Promise.all([
        api.get<any>("/admin/users?role=agent,controller"),
        api.get<any>("/admin/agencies"),
      ]);
      // Ensure we always set arrays to state, even if API returns non-array data
      const agentsArray = Array.isArray(agentsData) ? agentsData : [];
      const orgsArray = Array.isArray(orgsData) ? orgsData : [];
      // Map API snake_case to camelCase for isActive
      const mappedAgents = agentsArray.map(agent => ({
        ...agent,
        isActive: agent.is_active,
      }));
      setAgents(mappedAgents);
      setOrganizations(orgsArray);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  };

  const generatePin = () => {
    const pin = Math.floor(1000 + Math.random() * 9000).toString();
    setFormData({ ...formData, pin });
  };

  const handleSubmit = async (e: React.FormEvent) => {
    console.log("handleSubmit started");
    e.preventDefault();
    setFormLoading(true);
    setFormError(null);
    setFormSuccess(null);

    // Form validation
    const errors: string[] = [];
    const full_name = `${formData.first_name} ${formData.last_name}`.trim();
    if (!formData.first_name.trim()) {
      errors.push("Le prénom est requis");
    }
    if (!formData.last_name.trim()) {
      errors.push("Le nom est requis");
    }
    if (!formData.pin.trim()) {
      errors.push("Le PIN est requis");
    } else if (!/^\d{4}$/.test(formData.pin.trim())) {
      errors.push("Le PIN doit contenir exactement 4 chiffres");
    }
    if (formData.organizationIds.length === 0) {
      errors.push("Veuillez sélectionner au moins une organisation");
    }

    // Validate phone number format (exactly 12 digits) if provided
    if (
      formData.phone !== null &&
      formData.phone !== undefined &&
      formData.phone.trim() !== ""
    ) {
      if (formData.phone.trim().length !== 12) {
        errors.push("Le numéro de téléphone doit contenir exactement 12 chiffres");
      }
      // Check if it contains only digits
      if (!/^[0-9]+$/.test(formData.phone.trim())) {
        errors.push("Le numéro de téléphone doit contenir uniquement des chiffres");
      }
    }

    if (errors.length > 0) {
      console.log("Validation failed with errors:", errors);
      setFormError(errors.join("\n"));
      setFormLoading(false);
      return;
    }
    console.log("Validation successful");

    // Prepare payload for logging
    const payload = selectedAgent
      ? {
          full_name: full_name,
          phone: formData.phone,
          is_active: formData.is_active,
          role: formData.role,
          organization_ids: formData.organizationIds,
        }
      : {
          email: `${formData.first_name.toLowerCase()}.${formData.last_name.toLowerCase()}@ayaletrans.com`,
          password: formData.pin,
          full_name: full_name,
          phone: formData.phone,
          role: formData.role,
          is_active: formData.is_active,
          organization_ids: formData.organizationIds,
        };
    console.log("Payload to be sent:", payload);

    try {
      console.log("Starting API call");
      if (selectedAgent) {
        // Update existing agent
        await api.put(`/admin/users/${selectedAgent.id}`, {
          full_name: full_name,
          phone: formData.phone,
          is_active: formData.is_active,
          role: formData.role,
          organization_ids: formData.organizationIds,
        });
        console.log("Update API call successful");
        setSuccess("Agent mis à jour avec succès");
      } else {
        // Create new agent
        await api.post("/admin/users", {
          email: `${formData.first_name.toLowerCase()}.${formData.last_name.toLowerCase()}@ayaletrans.com`,
          password: formData.pin,
          full_name: full_name,
          phone: formData.phone,
          role: formData.role,
          is_active: formData.is_active,
          organization_ids: formData.organizationIds,
        });
        console.log("Create API call successful");
        setSuccess("Agent créé avec succès");
        // Optimistically add to list to see PIN immediately
        const newAgent: User = {
          id: Date.now().toString(), // temporary ID
          email: `${formData.first_name.toLowerCase()}.${formData.last_name.toLowerCase()}@ayaletrans.com`,
          full_name: full_name,
          phone: formData.phone ?? undefined,
          role: formData.role,
          organization_id: formData.organizationIds[0] ?? undefined,
          organizations: formData.organizationIds.map(id => ({ id, name: "" })) as Organization[],
          is_verified: false,
          created_at: new Date().toISOString(),
          pin: formData.pin,
          isActive: formData.is_active,
        };
        // Prepend new agent to top of list
        setAgents(prev => [newAgent, ...prev]);
      }
      resetForm();
      // Refresh the list after successful creation/update
      await fetchAll();
    } catch (err) {
      console.log("API call error:", err);
      let message = (err as Error).message;
      if (err && typeof err === "object" && "response" in err) {
        const axiosErr = err as any;
        const data = axiosErr.response.data;
        message = data?.error || data?.message || JSON.stringify(data);
      }
      setFormError(message);
    } finally {
      setFormLoading(false);
    }
  };

  const resetForm = () => {
    setFormData({
      first_name: "",
      last_name: "",
      phone: "",
      pin: "",
      role: "agent",
      organizationIds: [] as string[],
      is_active: true,
    });
    setSelectedAgent(null);
    setFormError(null);
    setFormSuccess(null);
  };

  const handleEdit = (agent: User) => {
    setSelectedAgent(agent);
    // Split full_name into first and last name (simple split on first space)
    const parts = agent.full_name.split(" ");
    const first_name = parts[0] || "";
    const last_name = parts.slice(1).join(" ") || "";
    setFormData({
      first_name: first_name,
      last_name: last_name,
      phone: agent.phone ?? "",
      pin: agent.pin ?? "",
      role: agent.role as "agent" | "controller",
      organizationIds: agent.organizations?.map(o => o.id) ?? [],
      is_active: agent.isActive, // Use the mapped isActive from User type
    });
  };

  const handleDelete = async (id: string) => {
    if (!window.confirm("Êtes-vous sûr de vouloir supprimer cet agent ?")) return;
    try {
      await api.delete(`/admin/users/${id}`);
      setSuccess("Agent supprimé avec succès");
      fetchAll();
    } catch (err) {
      setError((err as Error).message);
    }
  };

  const handleToggleActive = async (agent: User) => {
    // Optimistically update the agent's isActive status
    const updatedAgent = { ...agent, isActive: !agent.isActive };
    setAgents(prev =>
      prev.map(a => (a.id === agent.id ? updatedAgent : a))
    );
    try {
      await api.put(`/admin/users/${agent.id}`, {
        is_active: !agent.isActive,
      });
      // If successful, keep the optimistic update
      setSuccess(
        `Agent ${agent.isActive ? "désactivé" : "activé"} avec succès`
      );
    } catch (err) {
      // Revert optimistic update on error
      setAgents(prev =>
        prev.map(a => (a.id === agent.id ? agent : a))
      );
      console.error("Failed to toggle agent status:", err);
      let message = (err as Error).message;
      if (err && typeof err === "object" && "response" in err) {
        const axiosErr = err as any;
        const data = axiosErr.response.data;
        message = data?.error || data?.message || JSON.stringify(data);
      }
      setError(
        `Erreur lors du changement de statut : ${message}`
      );
    }
  };

  if (loading) {
    return <div className="text-center py-20 text-gray-500">Chargement...</div>;
  }

  // Compute filtered agents based on search term, organization, and status
  const filteredAgents = agents.filter(agent => {
    const matchesSearch = searchTerm
      ? (agent.full_name?.toLowerCase().includes(searchTerm.toLowerCase()) ||
          agent.email?.toLowerCase().includes(searchTerm.toLowerCase()) ||
          (agent.phone ?? '').toLowerCase().includes(searchTerm.toLowerCase()))
      : true;
    const matchesOrg = selectedOrgId
      ? agent.organization_id === selectedOrgId
      : true;
    const matchesStatus =
      selectedStatus === 'all'
        ? true
        : selectedStatus === 'active'
        ? agent.isActive
        : !agent.isActive;
    return matchesSearch && matchesOrg && matchesStatus;
  });

  return (
    <div className="space-y-6 max-w-7xl mx-auto p-6">
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-bold">Gestion des agents</h1>
        <div className="flex items-center gap-3">
          <button
            onClick={() => resetForm()}
            className="flex items-center gap-1 px-3 py-1.5 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 transition disabled:opacity-50"
          >
            <Plus className="h-4 w-4" /> Nouveau agent
          </button>
        </div>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg text-sm">
          {error}
        </div>
      )}

      {success && (
        <div className="bg-green-50 border border-green-200 text-green-700 px-4 py-3 rounded-lg text-sm">
          {success}
        </div>
      )}

      {/* Agent Form */}
      <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-100">
          <h2 className="font-semibold">
            {selectedAgent ? "Modifier l'agent" : "Nouvel agent"}
          </h2>
          <p className="text-xs text-gray-400 mt-0.5">
            {selectedAgent
              ? "Modifier les informations de l'agent sélectionné"
              : "Créer un nouvel agent"}
          </p>
        </div>

        {formError && (
          <div className="px-6 py-3 bg-red-50 border border-red-200 text-red-700 rounded-b-lg">
            {formError.split("\n").map((err, index) => (
              <div key={index} style={{ whiteSpace: "pre-line" }}>
                {err}
              </div>
            ))}
          </div>
        )}

        {formSuccess && (
          <div className="px-6 py-3 bg-green-50 border border-green-200 text-green-700 rounded-b-lg">
            {formSuccess}
          </div>
        )}

        <form onSubmit={handleSubmit} className="px-6 py-4 space-y-4">
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Prénom
              </label>
              <input
                type="text"
                value={formData.first_name}
                onChange={(e) =>
                  setFormData({ ...formData, first_name: e.target.value })
                }
                required
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Nom
              </label>
              <input
                type="text"
                value={formData.last_name}
                onChange={(e) =>
                  setFormData({ ...formData, last_name: e.target.value })
                }
                required
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Téléphone
              </label>
              <input
                type="tel"
                value={formData.phone}
                onChange={(e) =>
                  setFormData({ ...formData, phone: e.target.value })
                }
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Code PIN (4 chiffres)
              </label>
              <div className="flex items-center gap-2">
                <input
                  required
                  type="text"
                  value={formData.pin}
                  onChange={(e) =>
                    setFormData({ ...formData, pin: e.target.value })
                  }
                  className="flex-1 px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                </input>
                <button
                  type="button"
                  onClick={generatePin}
                  className="px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 bg-blue-50 hover:bg-blue-100"
                >
                  <RefreshCw className="h-4 w-4" />
                </button>
              </div>
            </div>
          </div>

          <div className="mt-4">
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Rôle
            </label>
            <div className="flex items-center gap-4">
              <div className="flex items-center gap-2">
                <input
                  type="radio"
                  value="agent"
                  checked={formData.role === "agent"}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      role: e.target.value as "agent" | "controller",
                    })
                  }
                  className="h-4 w-4 tcolor=blue-600"
                />
                <span className="text-sm font-medium text-gray-700">Agent</span>
              </div>
              <div className="flex items-center gap-2">
                <input
                  type="radio"
                  value="controller"
                  checked={formData.role === "controller"}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      role: e.target.value as "agent" | "controller",
                    })
                  }
                  className="h-4 w-4 text-blue-600"
                />
                <span className="text-sm font-medium text-gray-700">
                  Contrôleur
                </span>
              </div>
            </div>
          </div>

          <div className="mt-4">
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Organisation(s)
            </label>
            <select
              multiple
              value={formData.organizationIds}
              onChange={(e) => {
                const selectedValues = Array.from(e.target.selectedOptions).map(
                  option => option.value
                );
                setFormData({ ...formData, organizationIds: selectedValues });
              }}
            >
              {organizations.map(org => (
                <option key={org.id} value={org.id}>
                  {org.name}
                </option>
              ))}
            </select>
          </div>

          <div className="mt-4">
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Compte actif
            </label>
            <div className="flex items-center">
              <input
                type="checkbox"
                checked={formData.is_active}
                onChange={(e) =>
                  setFormData({ ...formData, is_active: e.target.checked })
                }
                className="h-4 w-4 text-blue-600"
              />
            </div>
          </div>

          {/* Submit button moved inside the form */}
          {selectedAgent ? (
            <div className="flex items-center gap-3">
              <button
                type="button"
                onClick={generatePin}
                className="flex items-center gap-1 px-3 py-1.5 text-sm font-medium text-white bg-gray-600 rounded-lg hover:bg-gray-700 transition"
              >
                <RefreshCw className="h-4 w-4" /> Régénérer PIN
              </button>
              <button
                type="submit"
                disabled={formLoading}
                className="flex items-center gap-1 px-3 py-1.5 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 transition disabled:opacity-50"
              >
                {formLoading ? "Enregistrement..." : "Mettre à jour l'agent"}
              </button>
            </div>
          ) : (
            <button
              type="submit"
              disabled={formLoading}
              className="w-full flex items-center gap-1 px-3 py-1.5 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 transition disabled:opacity-50"
            >
              {formLoading ? "Création..." : "Créer l'agent"}
            </button>
          )}
        </form>
      </div>
      {/* Agents List */}
      <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-100">
          <h2 className="font-semibold">Liste des agents</h2>
        </div>
        <div className="px-6 py-4">
          <div className="flex flex-wrap gap-4">
            <div className="flex-1 min-w-[200px]">
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Rechercher
              </label>
              <input
                type="text"
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                placeholder="Nom, email, téléphone"
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
            <div className="flex-1 min-w-[200px]">
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Agence
              </label>
              <select
                value={selectedOrgId ?? ''}
                onChange={(e) => {
                  const val = e.target.value;
                  setSelectedOrgId(val === '' ? null : val);
                }}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="">Toutes les agences</option>
                {organizations.map(org => (
                  <option key={org.id} value={org.id}>
                    {org.name}
                  </option>
                ))}
              </select>
            </div>
            <div className="flex-1 min-w-[200px]">
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Statut
              </label>
              <select
                value={selectedStatus}
                onChange={(e) => setSelectedStatus(e.target.value as 'all' | 'active' | 'inactive')}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="all">Tous</option>
                <option value="active">Actifs</option>
                <option value="inactive">Inactifs</option>
              </select>
            </div>
          </div>
        </div>

        {filteredAgents.length === 0 ? (
          <div className="px-6 py-8 text-center text-gray-500">
            Aucun agent trouvé
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full whitespace-nowrap">
              <thead>
                <tr className="bg-gray-50">
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Nom complet
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Email
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Téléphone
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-mono text-gray-500">
                    Code PIN
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Rôle
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Organisation(s)
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Actif
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {filteredAgents.map(agent => (
                  <tr key={agent.id} className="hover:bg-gray-50">
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                      {agent.full_name?.trim() || "-"}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm">
                      {agent.email ?? "-"}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {agent.phone || "-"}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-mono text-gray-500">
                      {agent.pin ?? "-"}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm">
                      <span
                        className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                          agent.role === "agent"
                            ? "bg-green-100 text-green-800"
                            : "bg-blue-100 text-blue-800"
                        }`}
                      >
                        {agent.role === "agent" ? "Agent" : "Contrôleur"}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm">
                      {agent.organizations && agent.organizations.length > 0 ? (
                        agent.organizations
                          .map(org => org.name)
                          .join(", ")
                      ) : (
                        "-"
                      )}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm">
                      <div className="flex items-center">
                        <input
                          type="checkbox"
                          checked={agent.isActive}
                          onChange={(e) => {
                            e.preventDefault();
                            handleToggleActive(agent);
                          }}
                          className="h-4 w-4 text-blue-600"
                        />
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium flex items-center gap-2">
                      <button
                        onClick={() => handleEdit(agent)}
                        className="p-1.5 rounded-md hover:bg-blue-50 hover:text-blue-600 transition-colors"
                        aria-label="Modifier l'agent"
                      >
                        <Edit2 className="h-4 w-4" />
                      </button>
                      <button
                        onClick={() => handleDelete(agent.id)}
                        className="p-1.5 rounded-md hover:bg-red-50 hover:text-red-600 transition-colors"
                        aria-label="Supprimer l'agent"
                      >
                        <Trash2 className="h-4 w-4" />
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}