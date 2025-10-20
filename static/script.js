        let fullSaveData;
        let originalFilename;
        const editor = document.getElementById('editor');
        const dropZone = document.getElementById('drop-zone');
        const saveFile = document.getElementById('saveFile');
        const saveChangesButton = document.getElementById('saveChangesButton');

        function loadSave() {
            const file = saveFile.files[0];
            handleFile(file);
        }

        function handleFile(file) {
            if (file) {
                originalFilename = file.name;
                const reader = new FileReader();
                reader.onload = function(e) {
                    const rawData = e.target.result;
                    fetch('/api/upload', {
                        method: 'POST',
                        body: rawData
                    })
                    .then(response => {
                        if (!response.ok) {
                            throw new Error('Network response was not ok');
                        }
                        return response.json();
                    })
                    .then(data => {
                        editor.textContent = JSON.stringify(data, null, 2);
                    })
                    .catch(error => {
                        console.error('Error uploading or processing file:', error);
                        alert('An error occurred while processing the file.');
                    });
                };
                reader.readAsText(file);
            } else {
                alert("Please select a valid save file.");
            }
        }

        window.addEventListener('dragenter', (event) => {
            event.preventDefault();
            dropZone.style.display = 'flex';
        });

        dropZone.addEventListener('dragover', (event) => {
            event.preventDefault();
        });

        dropZone.addEventListener('dragleave', (event) => {
            event.preventDefault();
            dropZone.style.display = 'none';
        });

        dropZone.addEventListener('drop', (event) => {
            event.preventDefault();
            dropZone.style.display = 'none';
            const file = event.dataTransfer.files[0];
            handleFile(file);
        });

        function saveChanges() {
            const text = editor.textContent;
            fetch('/api/save', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: text,
            })
            .then(response => response.json())
            .then(data => {
                document.getElementById('downloadLink').style.display = 'block';
                const saveFilename = originalFilename.replace(/\.[^/.]+$/, "") + ".save";
                document.getElementById('downloadUrl').href = '/api/download/' + data.id + '?filename=' + encodeURIComponent(saveFilename);
            })
            .catch((error) => {
                console.error('Error:', error);
                alert('An error occurred while saving the file.');
            });
        }

        // Attach event listeners
        saveFile.addEventListener('change', loadSave);
        saveChangesButton.addEventListener('click', saveChanges);