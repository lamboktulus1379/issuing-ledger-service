#!/bin/bash
# Quick deployment script - one command to rule them all!

echo "🚀 Quick Go Project Deployment"
echo "================================"

# Make scripts executable if they aren't already
chmod +x deploy-nginx.sh service.sh

# Run the deployment
./service.sh start

echo
echo "🎉 Deployment complete!"
echo
echo "Quick commands:"
echo "  ./service.sh status  # Check status"
echo "  ./service.sh test    # Run tests"
echo "  ./service.sh logs    # View logs"
echo "  ./service.sh stop    # Stop services"
echo
echo "Access your application:"
echo "  https://localhost:10010  (Direct)"
echo "  https://gra.tulus.tech   (Via proxy)"
echo "  http://localhost:10020   (Redirects to proxy)"
