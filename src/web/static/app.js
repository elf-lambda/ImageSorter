let selectedImage = null;
let allImages = [];
let lazyLoadObserver;

const previewFileNameP = document.getElementById("file-name");
const mainPreviewWrapper = document.getElementById("main-preview-wrapper");
const tagListElement = document.getElementById("tag-list");
const tagSearchInput = document.getElementById("tag-search");
const tagInput = document.getElementById("tag-input");
const addTagBtn = document.getElementById("add-tag-btn");
const copyFullUrlBtn = document.getElementById("copy-full-url-btn");
const copyFilenameBtn = document.getElementById("copy-filename-btn");

const clearChildren = (element) => {
    while (element.firstChild) {
        element.removeChild(element.firstChild);
    }
};

async function fetchImages() {
    try {
        const res = await fetch("/images-meta");
        if (!res.ok) {
            throw new Error(`HTTP error! status: ${res.status}`);
        }
        const images = await res.json();
        allImages = images;
        initializeLazyLoadObserver();
        renderImageList(images);
    } catch (error) {
        console.error("Failed to fetch images:", error);
        alert("Failed to load images. Please check the console for details.");
    }
}

// Initialize the Intersection Observer once
function initializeLazyLoadObserver() {
    // Disconnect existing observer if it was already initialized
    lazyLoadObserver?.disconnect(); // Optional chaining for cleaner disconnect

    lazyLoadObserver = new IntersectionObserver(
        (entries) => {
            entries.forEach((entry) => {
                if (entry.isIntersecting) {
                    const container = entry.target;
                    const mediaEl = container.querySelector("img");

                    if (
                        container.dataset.thumbnailUrl &&
                        !mediaEl.src &&
                        !container.dataset.loaded
                    ) {
                        container.dataset.loaded = "true";
                        mediaEl.src = container.dataset.thumbnailUrl;
                    }
                    lazyLoadObserver.unobserve(container);
                }
            });
        },
        {
            rootMargin: "50px",
            threshold: 0.1,
        }
    );
}

function filterImagesByTag(tag) {
    const lowerTag = tag.toLowerCase();
    const filteredImages = lowerTag
        ? allImages.filter(
              (img) =>
                  Array.isArray(img.Tags) &&
                  img.Tags.some((t) => t.toLowerCase().includes(lowerTag))
          )
        : allImages;
    renderImageList(filteredImages);
}

function renderImageList(images) {
    const list = document.getElementById("image-list");
    clearChildren(list);

    lazyLoadObserver?.disconnect();

    const fragment = document.createDocumentFragment();

    for (const img of images) {
        const div = document.createElement("div");
        div.className = "thumbnail";
        div.dataset.thumbnailUrl = img.ThumbnailURL;
        div.dataset.type = img.Name.toLowerCase().endsWith(".webm")
            ? "webm"
            : "image";

        const mediaEl = document.createElement("img");
        mediaEl.alt = img.Name;
        mediaEl.onerror = () => {
            mediaEl.src = "https://via.placeholder.com/150?text=Error";
        };

        if (div.dataset.type === "webm") {
            const playIcon = document.createElement("div");
            playIcon.innerHTML = "▶";
            playIcon.className = "play-icon";
            div.appendChild(playIcon);
        }

        div.onclick = () => showImageDetail(img);
        div.appendChild(mediaEl);
        fragment.appendChild(div);

        lazyLoadObserver.observe(div);
    }
    list.appendChild(fragment);
}

function showImageDetail(img) {
    selectedImage = img;
    clearChildren(mainPreviewWrapper);

    const isWebM = img.Name.toLowerCase().endsWith(".webm");
    let mediaElement;

    if (isWebM) {
        mediaElement = document.createElement("video");
        mediaElement.controls = true;
        mediaElement.loop = true;
        mediaElement.preload = "metadata";

        const playOverlay = document.createElement("div");
        playOverlay.innerHTML = "▶";
        playOverlay.className = "video-play-overlay";
        playOverlay.onclick = () => mediaElement.play();

        mediaElement.addEventListener("pause", () => {
            if (!mediaElement.ended) playOverlay.style.display = "flex";
        });
        mediaElement.addEventListener(
            "play",
            () => (playOverlay.style.display = "none")
        );
        mediaElement.addEventListener("loadedmetadata", () => {
            if (mediaElement.paused) playOverlay.style.display = "flex";
        });
        mediaElement.addEventListener(
            "ended",
            () => (playOverlay.style.display = "flex")
        );

        mainPreviewWrapper.appendChild(mediaElement);
        mainPreviewWrapper.appendChild(playOverlay);
    } else {
        mediaElement = document.createElement("img");
        mainPreviewWrapper.appendChild(mediaElement);
    }

    mediaElement.id = "image-preview";
    mediaElement.src = `/images/${img.Name}`;
    mediaElement.alt = "Preview";

    previewFileNameP.innerText = img.Name;
    renderTags(img.Tags || [], img.SHA256);
}

function renderTags(tags, sha) {
    clearChildren(tagListElement);

    const effectiveTags = Array.isArray(tags) ? tags : [];

    for (const tag of effectiveTags) {
        const span = document.createElement("span");
        span.className = "tag";
        span.textContent = tag;

        const button = document.createElement("button");
        button.textContent = "x";
        button.onclick = () => deleteTag(sha, tag);
        span.appendChild(button);
        tagListElement.appendChild(span);
    }
}

async function handleTagAction(action, sha, tag) {
    if (!sha || !tag) return;

    console.log(`${action}ing tag: [${tag}] for SHA: ${sha}`);
    try {
        const response = await fetch(
            `/${action}-tag?sha=${sha}&tag=${encodeURIComponent(tag)}`,
            { method: "POST" }
        );

        if (response.ok) {
            if (action === "add") tagInput.value = "";
            await refreshImageTags(sha);
        } else {
            const errorText = await response.text();
            console.error(
                `Failed to ${action} tag:`,
                response.status,
                response.statusText,
                errorText
            );
            alert(`Failed to ${action} tag. See console for details.`);
        }
    } catch (error) {
        console.error(`Error during ${action} tag operation:`, error);
        alert(`An error occurred while trying to ${action} the tag.`);
    }
}

const addTag = () =>
    handleTagAction("add", selectedImage?.SHA256, tagInput.value.trim());

const deleteTag = (sha, tag) => handleTagAction("delete", sha, tag);

async function refreshImageTags(sha) {
    await fetchImages(); // Re-fetch all images to update allImages
    const found = allImages.find((img) => img.SHA256 === sha);
    if (found) {
        showImageDetail(found);
    }
}

async function copyToClipboard(text) {
    try {
        await navigator.clipboard.writeText(text);
        console.log("Copied to clipboard:", text);
        showCopyMessage("Copied!");
    } catch (err) {
        console.error("Failed to copy text: ", err);
        alert(
            "Failed to copy. Your browser might require HTTPS or user gesture for clipboard access."
        );
    }
}

function showCopyMessage(message) {
    const messageDiv = document.createElement("div");
    messageDiv.textContent = message;
    messageDiv.style.cssText = `
        position: fixed;
        bottom: 20px;
        right: 20px;
        background-color: #4CAF50;
        color: white;
        padding: 10px 15px;
        border-radius: 5px;
        z-index: 9999;
        opacity: 0;
        transition: opacity 0.5s ease-in-out;
    `;
    document.body.appendChild(messageDiv);
    setTimeout(() => {
        messageDiv.style.opacity = 1;
    }, 10);
    setTimeout(() => {
        messageDiv.style.opacity = 0;
        messageDiv.addEventListener("transitionend", () => messageDiv.remove());
    }, 1500);
}

function renameAllFiles() {
    if (
        !confirm(
            "Are you sure you want to rename all files by SHA256? This cannot be undone."
        )
    )
        return;
    fetch("/rename-all", { method: "POST" })
        .then((res) => {
            if (!res.ok) throw new Error("Rename failed");
            return fetchImages(); // Reload images after rename
        })
        .catch((err) => {
            console.error(err);
            alert("Rename failed. See console for details.");
        });
}

window.onload = () => {
    fetchImages();

    addTagBtn.onclick = addTag;
    tagSearchInput.addEventListener("input", (e) => {
        filterImagesByTag(e.target.value.trim());
    });

    copyFullUrlBtn.onclick = () => {
        if (selectedImage) {
            const fullUrl = `C:/Users/elff/Pictures/${selectedImage.Name}`;
            copyToClipboard(fullUrl);
        } else {
            alert("Please select an image first.");
        }
    };

    copyFilenameBtn.onclick = () => {
        if (selectedImage) {
            copyToClipboard(selectedImage.Name);
        } else {
            alert("Please select an image first.");
        }
    };
};
