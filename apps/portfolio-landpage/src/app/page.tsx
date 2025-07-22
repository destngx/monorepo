'use client';

import React from 'react';
import './page.css';

if (typeof window !== 'undefined') {
  window.history.scrollRestoration = 'manual';
}


const Home = () => {
    return (
      <div className="portfolio-container">
        <header className="header">
          <div className="header-content">
            <h1>Quang Dinh</h1>
            <h2>DevOps Engineer</h2>
            <nav className="nav">
              <a href="#skills">Skills</a>
              <a href="#projects">Projects</a>
              <a href="#stories">Stories</a>
              <a href="#contact">Contact</a>
            </nav>
          </div>
        </header>

        <section className="hero">
          <div className="hero-content">
            <h2>Hi, I&apos;m Quang Dinh</h2>
            <p>
              DevOps Engineer at Nexon Dev Vina. I build and maintain scalable infrastructure and CI/CD pipelines on AWS.
              Transitioned from backend engineering to DevOps, passionate about optimizing deployments, enabling fast feedback loops, and ensuring secure, reliable systems.
            </p>
            <a href="#contact" className="cta-btn">Contact Me</a>
          </div>
        </section>

        <section className="skills" id="skills">
          <h2>Skills</h2>
          <ul className="skills-list">
            <li>CI/CD: GitLab CI/CD, GitHub Actions, Jenkins</li>
            <li>Cloud: AWS (EC2, monitoring, scalable app deployment)</li>
            <li>Containers: Docker, docker-compose, Kubernetes + GitOps</li>
            <li>Monitoring: Logging/tracing, Prometheus-style observability</li>
            <li>Security: Web security inspection, IAM</li>
            <li>IaC: Terraform (basic experience)</li>
            <li>Load Balancer: NGINX (reverse proxy, scaling)</li>
            <li>Collaboration: Slack</li>
            <li>Willingness to learn: Grafana, Prometheus, HAProxy</li>
          </ul>
        </section>

        <section className="projects" id="projects">
          <h2>Projects</h2>
          <div className="projects-list">
            <div className="project-card">
              <h3>Internal Attendance Platform</h3>
              <p>Backend system for tracking employee attendance.</p>
              <span>Technologies: Node.js, PostgreSQL</span>
            </div>
            <div className="project-card">
              <h3>Content Scanner (ML)</h3>
              <p>Machine learning-based content scanner for internal use.</p>
              <span>Technologies: Python, ML, AWS</span>
            </div>
            <div className="project-card">
              <h3>GitLab CI/CD for Lambda@Edge</h3>
              <p>Automated deployment pipeline, reduced deployment time from 30+ min to under 5 min.</p>
              <span>Technologies: GitLab CI/CD, AWS Lambda, CloudFront</span>
            </div>
            <div className="project-card">
              <h3>Kubernetes GitOps Infrastructure</h3>
              <p>Managed clusters with ArgoCD, enabled fast incident response and onboarding.</p>
              <span>Technologies: Kubernetes, ArgoCD, GitOps</span>
            </div>
            <div className="project-card">
              <h3>Redis Migration</h3>
              <p>Migrated Redis workloads to AWS ElastiCache for scalable performance.</p>
              <span>Technologies: Redis, AWS ElastiCache</span>
            </div>
          </div>
        </section>

        <section className="blog" id="stories">
          <h2>Stories & Blog</h2>
          <div className="blog-list">
            <div className="blog-card">
              <h3>Securing CI/CD Pipelines in GitLab</h3>
              <p>Multi-layered protection: network restrictions, SSO, least privilege, secrets management, manual approvals.</p>
            </div>
            <div className="blog-card">
              <h3>Redis Scaling and Migration</h3>
              <p>How we scaled Redis with AWS ElastiCache and solved connection pool contention in Bull queue.</p>
            </div>
            <div className="blog-card">
              <h3>Incident Response with GitOps & ArgoCD</h3>
              <p>Declarative infrastructure, version control, fast rollback, and onboarding.</p>
            </div>
            <div className="blog-card">
              <h3>Automated Lambda@Edge Deployment</h3>
              <p>CI/CD pipeline for Lambda@Edge, reducing deployment time and improving developer confidence.</p>
            </div>
          </div>
        </section>

        <section className="certifications" id="certifications">
          <h2>Certifications & Achievements</h2>
          <ul>
            <li>Preparing for AWS/CKAD certifications</li>
            <li>Led cross-functional projects</li>
            <li>Improved deployment times by 50%</li>
            <li>Managed high-traffic infrastructure</li>
          </ul>
        </section>

        <section className="contact" id="contact">
          <h2>Contact</h2>
          <div className="contact-info">
            <p>Email: your.email@example.com</p>
            <p>
              <a href="https://linkedin.com/in/yourprofile" target="_blank">LinkedIn</a> |
              <a href="https://github.com/yourgithub" target="_blank">GitHub</a>
            </p>
          </div>
          <form className="contact-form">
            <label>
              Name:
              <input type="text" name="name" />
            </label>
            <label>
              Email:
              <input type="email" name="email" />
            </label>
            <label>
              Message:
              <textarea name="message"></textarea>
            </label>
            <button type="submit">Send</button>
          </form>
        </section>

        <footer className="footer">
          <p>© Quang Dinh</p>
          <p>
            <a href="https://linkedin.com/in/yourprofile" target="_blank">LinkedIn</a> |
            <a href="https://github.com/yourgithub" target="_blank">GitHub</a>
          </p>
        </footer>
      </div>
    );
};

export default Home;
