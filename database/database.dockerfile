# Use MongoDB image
FROM mongo:7.0

# Expose port 27017
EXPOSE 27017

# Command to run MongoDB
CMD ["mongod", "--bind_ip_all"]
