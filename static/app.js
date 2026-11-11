/**
 * NotifyEd - Cliente Frontend para la API REST
 * Consume los endpoints de materias (/courses) del servidor Go:
 *   - GET    /courses       (Listar todas las materias)
 *   - POST   /courses       (Crear una nueva materia)
 *   - DELETE /courses/{id}  (Eliminar una materia por ID)
 */

document.addEventListener("DOMContentLoaded", () => {
    // =========================================================================
    // Configuración de la API REST
    // =========================================================================
    // Si la web se sirve desde el mismo servidor Go (:8080), usamos URL relativa.
    // Si se abre con un servidor estático (ej: Live Server en :5500 o :3000),
    // apuntamos al host completo http://localhost:8080.
    const API_BASE_URL = window.location.origin.includes(":8080")
        ? ""
        : "http://localhost:8080";

    const ENDPOINT_COURSES = `${API_BASE_URL}/courses`;

    // =========================================================================
    // Referencias al DOM
    // =========================================================================
    const entityForm = document.getElementById("entity-form");
    const courseNameInput = document.getElementById("course-name");
    const charCounter = document.getElementById("char-counter");
    const formError = document.getElementById("form-error");
    const submitBtn = document.getElementById("submit-btn");
    const btnText = submitBtn.querySelector(".btn-text");
    const btnSpinner = submitBtn.querySelector(".btn-spinner");

    const entitiesList = document.getElementById("entities-list");
    const entityCountBadge = document.getElementById("entity-count-badge");
    const loadingIndicator = document.getElementById("loading-indicator");
    const connectionError = document.getElementById("connection-error");
    const emptyState = document.getElementById("empty-state");
    const refreshBtn = document.getElementById("refresh-btn");
    const retryBtn = document.getElementById("retry-btn");
    const toast = document.getElementById("toast");

    let toastTimer = null;

    // =========================================================================
    // Funciones Auxiliares de Interfaz (Toast, Errores, Escapado)
    // =========================================================================

    /**
     * Escapa caracteres especiales de HTML para prevenir ataques XSS al renderizar.
     */
    function escapeHTML(str) {
        if (!str) return "";
        return str
            .replace(/&/g, "&amp;")
            .replace(/</g, "&lt;")
            .replace(/>/g, "&gt;")
            .replace(/"/g, "&quot;")
            .replace(/'/g, "&#039;");
    }

    /**
     * Muestra una notificación temporal tipo Toast en pantalla.
     */
    function showToast(message, type = "success") {
        if (!toast) return;

        clearTimeout(toastTimer);
        toast.textContent = message;
        toast.className = `toast toast-${type}`;

        toastTimer = setTimeout(() => {
            toast.className = "toast hidden";
        }, 3500);
    }

    /**
     * Muestra un mensaje de error en el formulario.
     */
    function showFormError(message) {
        formError.textContent = message;
        formError.classList.remove("hidden");
        courseNameInput.classList.add("input-error");
    }

    /**
     * Limpia el mensaje de error del formulario.
     */
    function clearFormError() {
        formError.textContent = "";
        formError.classList.add("hidden");
        courseNameInput.classList.remove("input-error");
    }

    /**
     * Actualiza el contador de caracteres del campo de texto.
     */
    courseNameInput.addEventListener("input", () => {
        clearFormError();
        const currentLength = courseNameInput.value.length;
        const maxLength = courseNameInput.maxLength || 40;
        charCounter.textContent = `${currentLength} / ${maxLength}`;

        if (currentLength >= maxLength) {
            charCounter.classList.add("limit-reached");
        } else {
            charCounter.classList.remove("limit-reached");
        }
    });

    // =========================================================================
    // 1. Obtener y Renderizar Entidades (GET /courses)
    // =========================================================================

    /**
     * Realiza una petición GET a la API y refresca el listado en el DOM.
     */
    async function loadCourses() {
        // Mostrar estado de carga y ocultar estados previos
        loadingIndicator.classList.remove("hidden");
        entitiesList.classList.add("hidden");
        emptyState.classList.add("hidden");
        connectionError.classList.add("hidden");

        try {
            const response = await fetch(ENDPOINT_COURSES, {
                method: "GET",
                headers: {
                    "Accept": "application/json"
                }
            });

            if (!response.ok) {
                throw new Error(`Error en el servidor: HTTP ${response.status}`);
            }

            const courses = await response.json();
            renderCourses(Array.isArray(courses) ? courses : []);
        } catch (error) {
            console.error("Error al cargar las materias:", error);
            loadingIndicator.classList.add("hidden");
            connectionError.classList.remove("hidden");
            entityCountBadge.textContent = "0";
            showToast("No se pudo cargar la lista de materias.", "error");
        }
    }

    /**
     * Renderiza dinámicamente los elementos <li> dentro del contenedor del DOM.
     * Cada elemento muestra el nombre, ID y el botón para eliminar.
     */
    function renderCourses(courses) {
        loadingIndicator.classList.add("hidden");
        entityCountBadge.textContent = courses.length.toString();

        if (courses.length === 0) {
            emptyState.classList.remove("hidden");
            entitiesList.classList.add("hidden");
            entitiesList.innerHTML = "";
            return;
        }

        emptyState.classList.add("hidden");
        entitiesList.innerHTML = "";

        courses.forEach((course) => {
            const li = document.createElement("li");
            li.className = "entity-item";
            li.dataset.id = course.id_course;

            li.innerHTML = `
                <div class="entity-info">
                    <span class="entity-name">${escapeHTML(course.name)}</span>
                    <span class="entity-id" title="Identificador UUID">ID: ${escapeHTML(course.id_course)}</span>
                </div>
                <div class="entity-actions">
                    <button
                        type="button"
                        class="btn-delete"
                        data-id="${escapeHTML(course.id_course)}"
                        data-name="${escapeHTML(course.name)}"
                        aria-label="Eliminar ${escapeHTML(course.name)}"
                    >
                        🗑️ Eliminar
                    </button>
                </div>
            `;

            entitiesList.appendChild(li);
        });

        entitiesList.classList.remove("hidden");
    }

    // =========================================================================
    // 2. Crear una Nueva Entidad (POST /courses)
    // =========================================================================

    entityForm.addEventListener("submit", async (event) => {
        // Prevenir el comportamiento por defecto de recarga del formulario
        event.preventDefault();
        clearFormError();

        const name = courseNameInput.value.trim();

        // Validaciones del lado del cliente
        if (!name) {
            showFormError("Por favor ingresa un nombre para la materia.");
            courseNameInput.focus();
            return;
        }

        if (name.length > 40) {
            showFormError("El nombre no puede exceder los 40 caracteres.");
            courseNameInput.focus();
            return;
        }

        // Estado visual del botón durante el envío
        submitBtn.disabled = true;
        btnText.textContent = "Guardando...";
        btnSpinner.classList.remove("hidden");

        try {
            const response = await fetch(ENDPOINT_COURSES, {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                    "Accept": "application/json"
                },
                body: JSON.stringify({ name })
            });

            if (!response.ok) {
                let errorMessage = `Error al crear la materia (HTTP ${response.status})`;
                try {
                    const errData = await response.json();
                    if (errData && errData.error) {
                        errorMessage = errData.error;
                    }
                } catch {
                    // Fallback a mensaje por defecto
                }
                throw new Error(errorMessage);
            }

            // Éxito: limpiar formulario y notificar
            entityForm.reset();
            charCounter.textContent = "0 / 40";
            charCounter.classList.remove("limit-reached");
            showToast(`Materia "${name}" creada exitosamente.`, "success");

            // Volver a pedir la lista completa para refrescar la vista
            await loadCourses();
        } catch (error) {
            console.error("Error al crear materia:", error);
            showFormError(error.message);
            showToast(error.message, "error");
        } finally {
            // Restaurar estado del botón
            submitBtn.disabled = false;
            btnText.textContent = "Crear Materia";
            btnSpinner.classList.add("hidden");
        }
    });

    // =========================================================================
    // 3. Implementación del Borrado con Delegación de Eventos (DELETE /courses/{id})
    // =========================================================================

    /**
     * Delegación de eventos en el contenedor padre (entitiesList).
     * Como los botones se crean dinámicamente, escuchamos los clics en el contenedor.
     */
    entitiesList.addEventListener("click", async (event) => {
        const deleteBtn = event.target.closest(".btn-delete");
        if (!deleteBtn) return;

        const id = deleteBtn.dataset.id;
        const name = deleteBtn.dataset.name || "esta materia";

        if (!id) return;

        // Cuadro de confirmación antes de eliminar
        const confirmed = window.confirm(`¿Estás seguro de que deseas eliminar la materia "${name}"?`);
        if (!confirmed) return;

        // Deshabilitar botón durante la petición
        deleteBtn.disabled = true;
        deleteBtn.textContent = "Borrando...";

        try {
            const response = await fetch(`${ENDPOINT_COURSES}/${id}`, {
                method: "DELETE",
                headers: {
                    "Accept": "application/json"
                }
            });

            if (!response.ok) {
                let errorMessage = `No se pudo eliminar la materia (HTTP ${response.status})`;
                try {
                    const errData = await response.json();
                    if (errData && errData.error) {
                        errorMessage = errData.error;
                    }
                } catch {
                    // Fallback
                }
                throw new Error(errorMessage);
            }

            // Éxito: notificar y refrescar la lista completa
            showToast(`Materia "${name}" eliminada con éxito.`, "success");
            await loadCourses();
        } catch (error) {
            console.error("Error al eliminar la materia:", error);
            showToast(error.message, "error");
            // Restaurar el botón si falló
            deleteBtn.disabled = false;
            deleteBtn.innerHTML = "🗑️ Eliminar";
        }
    });

    // =========================================================================
    // 4. Eventos Adicionales de Control (Refrescar y Reintentar)
    // =========================================================================
    if (refreshBtn) {
        refreshBtn.addEventListener("click", () => {
            loadCourses();
        });
    }

    if (retryBtn) {
        retryBtn.addEventListener("click", () => {
            loadCourses();
        });
    }

    // Carga inicial al iniciar la aplicación
    loadCourses();
});

