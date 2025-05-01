FROM eclipse-temurin:17-jre

WORKDIR /usr/src/saxon

### download Saxon

# Install unzip to extract the Saxon-HE zip file
RUN apt-get update && apt-get install -y unzip && rm -rf /var/lib/apt/lists/*

# Download full Saxon-HE distribution (zip)
RUN curl -L -o SaxonHE12-5J.zip https://github.com/Saxonica/Saxon-HE/releases/download/SaxonHE12-5/SaxonHE12-5J.zip
RUN unzip SaxonHE12-5J.zip -d SaxonHE12-5J
RUN rm SaxonHE12-5J.zip

# Set the entrypoint to use Saxon-HE.jar directly
ENTRYPOINT ["java", "-jar", "SaxonHE12-5J/saxon-he-12.5.jar"]

