'use client';

import React from 'react';
import './page.css';

if (typeof window !== 'undefined') {
  window.history.scrollRestoration = 'manual';
}

import { useState, useEffect } from 'react';
import {
  Menu,
  X,
  ArrowDown,
  Github,
  Linkedin,
  Mail,
  Code2,
  Cloud,
  Shield,
  TrendingUp,
  Server,
  Wrench,
  Database,
  Gauge,
  Briefcase,
  Calendar,
  ExternalLink,
  Music,
  Gamepad2,
  PersonStanding,
  Bike,
  Swords,
  BookOpen,
  MapPin,
  Phone,
} from 'lucide-react';
import Image from 'next/image';

export default function Home() {
  // Navigation state
  const [isScrolled, setIsScrolled] = useState(false);
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);

  // Contact form state
  const [formData, setFormData] = useState({
    name: '',
    email: '',
    message: '',
  });

  useEffect(() => {
    const handleScroll = () => {
      setIsScrolled(window.scrollY > 50);
    };
    window.addEventListener('scroll', handleScroll);
    return () => window.removeEventListener('scroll', handleScroll);
  }, []);

  const scrollToSection = (href: string) => {
    const element = document.querySelector(href);
    if (element) {
      element.scrollIntoView({ behavior: 'smooth' });
      setIsMobileMenuOpen(false);
    }
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    // console.log("Form submitted:", formData)
  };

  const navItems = [
    { name: 'About', href: '#about' },
    { name: 'Skills', href: '#skills' },
    { name: 'Experience', href: '#experience' },
    { name: 'Projects', href: '#projects' },
    { name: 'Hobbies', href: '#hobbies' },
    { name: 'Contact', href: '#contact' },
  ];

  const aboutCards = [
    {
      icon: Cloud,
      title: 'Cloud Infrastructure Expert',
      description:
        'Specializing in building and maintaining robust cloud infrastructure on AWS. With expertise in CI/CD pipelines, Kubernetes orchestration, and infrastructure as code, I help teams deliver software faster and more reliably.',
    },
    {
      icon: TrendingUp,
      title: 'Performance & Cost Optimization',
      description:
        'Track record of reducing deployment costs by 50% while maintaining 99.9% uptime through strategic resource optimization, automated scaling, and fault-tolerant system design. Managing GitLab CI/CD pipelines for 10+ projects and overseeing multiple Kubernetes clusters using GitOps practices.',
    },
    {
      icon: Shield,
      title: 'Security & Monitoring',
      description:
        'Passionate about implementing monitoring solutions with Grafana, Prometheus, and Loki, ensuring optimal application performance and proactive incident detection. Focus on security best practices with regular inspections and automated security checks in CI/CD pipelines.',
    },
    {
      icon: Code2,
      title: 'Education & Passion',
      description:
        "Graduated from the University of Science - VNUHCM with a Bachelor's degree in Computer Science, specializing in Software Development. Continuously exploring new DevOps tools and contributing to the tech community.",
    },
  ];

  const skillCategories = [
    {
      icon: Cloud,
      title: 'Cloud & Infrastructure',
      skills: ['AWS', 'Terraform', 'Docker', 'Kubernetes', 'GitOps', 'NGINX'],
    },
    {
      icon: Gauge,
      title: 'CI/CD & Automation',
      skills: ['GitLab CI', 'GitHub Actions', 'Jenkins', 'Cost Optimization', 'Auto-scaling', 'Zero-downtime Deploy'],
    },
    {
      icon: Server,
      title: 'Monitoring & Observability',
      skills: ['Prometheus', 'Grafana', 'Loki', 'Slack Alerts', 'Performance Monitoring', 'Security Scanning'],
    },
    {
      icon: Code2,
      title: 'Programming Languages',
      skills: ['JavaScript', 'TypeScript', 'Python', 'Shell Script', 'Java', 'C/C++'],
    },
    {
      icon: Database,
      title: 'Backend & Databases',
      skills: ['NestJS', 'Node.js', 'PostgreSQL', 'MongoDB', 'MySQL', 'REST APIs'],
    },
    {
      icon: Wrench,
      title: 'Development Tools',
      skills: ['Git', 'Linux', 'macOS', 'React', 'Next.js', 'VS Code'],
    },
  ];

  const experiences = [
    {
      title: 'DevOps Engineer',
      company: 'Tech Company',
      period: '2024 - Present',
      description:
        'Managing GitLab CI/CD pipelines for 10+ projects and AWS cloud infrastructure. Achieved 50% cost reduction while maintaining 99.9% uptime through strategic optimization. Implemented comprehensive monitoring with Grafana, Prometheus, and Loki. Managing multiple Kubernetes clusters using GitOps for robust software development experiences.',
      technologies: ['AWS', 'Kubernetes', 'GitLab CI', 'Terraform', 'Grafana', 'Prometheus', 'GitOps'],
    },
    {
      title: 'Team Leader',
      company: 'Tech Company',
      period: '2023',
      description:
        'Led five projects from initiation to successful deployment, managing workflows for 10 teams. Analyzed requirements, divided tasks, and handled risk management. Developed serverless applications for document management with OpenSearch integration and user feedback analysis using text tokenization.',
      technologies: ['OpenSearch', 'AWS Lambda', 'NestJS', 'Team Management', 'Agile'],
    },
    {
      title: 'Back-End Developer',
      company: 'Tech Company',
      period: 'April 2022 - 2023',
      description:
        'Designed and developed an attendance system for 400 employees using clean architecture and microservices patterns with NestJS. Built ML-powered application for harmful content detection using Django and YoloV7. Optimized Docker builds and implemented zero-downtime deployment, reducing build and deploy time by 50%.',
      technologies: ['NestJS', 'Django', 'Docker', 'PostgreSQL', 'Machine Learning', 'YoloV7'],
    },
  ];

  const projects = [
    {
      title: 'AWS Cloud Infrastructure',
      description:
        'Designed and managed scalable AWS infrastructure capable of handling hundreds of thousands of requests per second. Implemented cost optimization strategies reducing deployment costs by 50% while maintaining 99.9% uptime through automated scaling and fault-tolerant design.',
      image: '/images/aws-cloud-infrastructure-dashboard.jpg',
      technologies: ['AWS', 'Terraform', 'Auto-scaling', 'Load Balancing'],
      liveUrl: '#',
      githubUrl: '#',
    },
    {
      title: 'Kubernetes GitOps Platform',
      description:
        'Managed multiple Kubernetes clusters using GitOps practices for robust and easy maintenance. Implemented automated deployment pipelines with GitLab CI/CD for 10+ projects, ensuring consistent and reliable software delivery.',
      image: '/images/kubernetes-gitops-dashboard.jpg',
      technologies: ['Kubernetes', 'GitOps', 'GitLab CI', 'Helm', 'ArgoCD'],
      liveUrl: '#',
      githubUrl: '#',
    },
    {
      title: 'Monitoring & Observability Stack',
      description:
        'Implemented comprehensive monitoring solution using Grafana, Prometheus, and Loki for application state observation and server monitoring. Set up proactive alerting system for incident detection and troubleshooting, ensuring optimal performance.',
      image: '/images/grafana-prometheus-monitoring-dashboard.jpg',
      technologies: ['Grafana', 'Prometheus', 'Loki', 'Slack', 'Alerting'],
      liveUrl: '#',
      githubUrl: '#',
    },
    {
      title: 'Employee Attendance System',
      description:
        'Designed and developed an attendance system used by 400 employees, automating requests and workforce management. Built with clean architecture, domain-driven design, and microservices pattern using NestJS for scalability and maintainability.',
      image: '/images/employee-attendance-management-system.jpg',
      technologies: ['NestJS', 'Microservices', 'PostgreSQL', 'Docker'],
      liveUrl: '#',
      githubUrl: '#',
    },
    {
      title: 'Document Management Platform',
      description:
        'Developed serverless application for company document management with OpenSearch integration for powerful search capabilities. Implemented user feedback analysis using text tokenization to categorize issues and improve user experience.',
      image: '/images/document-management-search-platform.jpg',
      technologies: ['AWS Lambda', 'OpenSearch', 'NestJS', 'Serverless'],
      liveUrl: '#',
      githubUrl: '#',
    },
    {
      title: 'ML Content Detection System',
      description:
        'Built machine learning application to detect harmful hand signs and predict inappropriate images using Django and YoloV7. Implemented automated content moderation pipeline for enhanced platform safety and user protection.',
      image: '/images/machine-learning-content-detection.jpg',
      technologies: ['Django', 'YoloV7', 'Machine Learning', 'Python'],
      liveUrl: '#',
      githubUrl: '#',
    },
  ];

  const hobbies = [
    {
      name: 'Bass Guitar',
      icon: Music,
      description: 'Creating grooves and rhythms',
    },
    {
      name: 'Gamification',
      icon: Gamepad2,
      description: 'Turning challenges into games',
    },
    {
      name: 'Running',
      icon: PersonStanding,
      description: 'Pushing limits one step at a time',
    },
    {
      name: 'Cycling',
      icon: Bike,
      description: '200km & 300km pendants achieved',
    },
    {
      name: 'Legend of Zelda',
      icon: Swords,
      description: "Exploring Hyrule's adventures",
    },
    {
      name: 'Manga',
      icon: BookOpen,
      description: 'Diving into visual storytelling',
    },
  ];

  return (
    <main className="relative min-h-screen">
      {/* Floating Shapes Background */}
      <div className="fixed inset-0 overflow-hidden pointer-events-none">
        <div className="absolute top-20 left-10 w-72 h-72 bg-primary/5 rounded-full blur-3xl animate-float" />
        <div
          className="absolute top-40 right-20 w-96 h-96 bg-accent/5 rounded-full blur-3xl animate-float"
          style={{ animationDelay: '2s' }}
        />
        <div
          className="absolute bottom-20 left-1/4 w-80 h-80 bg-primary/5 rounded-full blur-3xl animate-float"
          style={{ animationDelay: '4s' }}
        />
      </div>

      {/* Navigation */}
      <nav
        className={`fixed top-0 left-0 right-0 z-50 transition-all duration-300 ${
          isScrolled ? 'glass-strong py-4' : 'py-6'
        }`}
      >
        <div className="container mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between">
            <a
              href="#"
              className="text-xl font-bold tracking-tight text-foreground hover:text-primary transition-colors"
            >
              destnguyxn
            </a>

            <div className="hidden md:flex items-center gap-8">
              {navItems.map((item) => (
                <button
                  key={item.name}
                  onClick={() => scrollToSection(item.href)}
                  className="text-sm font-medium text-muted-foreground hover:text-foreground transition-colors"
                >
                  {item.name}
                </button>
              ))}
            </div>

            <button
              className="md:hidden inline-flex items-center justify-center rounded-md p-2 text-foreground hover:bg-accent/10 transition-colors"
              onClick={() => setIsMobileMenuOpen(!isMobileMenuOpen)}
            >
              {isMobileMenuOpen ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
            </button>
          </div>
        </div>
      </nav>

      {/* Mobile Menu */}
      {isMobileMenuOpen && (
        <div className="fixed inset-0 z-40 md:hidden">
          <div
            className="absolute inset-0 bg-background/80 backdrop-blur-sm"
            onClick={() => setIsMobileMenuOpen(false)}
          />
          <div className="absolute top-20 left-4 right-4 glass-strong rounded-2xl p-6 space-y-4">
            {navItems.map((item) => (
              <button
                key={item.name}
                onClick={() => scrollToSection(item.href)}
                className="block w-full text-left text-lg font-medium text-foreground hover:text-primary transition-colors py-2"
              >
                {item.name}
              </button>
            ))}
          </div>
        </div>
      )}

      <div className="relative z-10">
        {/* Hero Section */}
        <section className="min-h-screen flex items-center justify-center px-4 sm:px-6 lg:px-8">
          <div className="container mx-auto max-w-6xl">
            <div className="glass rounded-3xl p-8 sm:p-12 lg:p-16">
              <div className="flex flex-col lg:flex-row items-center gap-8 lg:gap-12">
                <div className="flex-shrink-0">
                  <div className="relative w-48 h-48 sm:w-56 sm:h-56 lg:w-64 lg:h-64">
                    <div className="absolute inset-0 bg-gradient-to-br from-primary/20 to-accent/20 rounded-2xl blur-xl" />
                    <div className="relative w-full h-full rounded-2xl overflow-hidden border-2 border-primary/10 shadow-2xl">
                      <Image src="/images/me.jpg" alt="Quang Dinh Nguyen Pham" fill className="object-cover" priority />
                    </div>
                  </div>
                </div>

                <div className="flex-1 text-center lg:text-left">
                  <h1 className="text-4xl sm:text-5xl lg:text-6xl font-bold tracking-tight mb-4 text-balance">
                    <span className="bg-gradient-to-r from-primary to-accent bg-clip-text text-transparent">
                      Quang Dinh Nguyen Pham
                    </span>
                  </h1>

                  <p className="text-xl sm:text-2xl text-foreground/80 mb-4 font-medium text-balance">
                    DevOps Software Engineer
                  </p>

                  <p className="text-base text-muted-foreground mb-6 leading-relaxed text-pretty">
                    Specializing in AWS, Kubernetes, CI/CD, cloud infrastructure optimization, and high availability
                    solutions. Passionate about building scalable systems and automating workflows.
                  </p>

                  <div className="flex flex-wrap items-center justify-center lg:justify-start gap-4 mb-8">
                    <a
                      href="#contact"
                      className="inline-flex items-center justify-center rounded-lg bg-accent hover:bg-accent/90 text-white px-6 py-3 text-base font-medium transition-colors"
                    >
                      <Mail className="mr-2 h-4 w-4" />
                      Get in Touch
                    </a>
                    <a
                      href="#projects"
                      className="inline-flex items-center justify-center rounded-lg glass hover:glass-strong bg-transparent border border-border/50 px-6 py-3 text-base font-medium transition-colors"
                    >
                      View Projects
                    </a>
                  </div>

                  <div className="flex items-center justify-center lg:justify-start gap-6">
                    <a
                      href="https://github.com/destnguyxn"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-muted-foreground hover:text-foreground transition-colors"
                    >
                      <Github className="h-6 w-6" />
                      <span className="sr-only">GitHub</span>
                    </a>
                    <a
                      href="https://linkedin.com"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-muted-foreground hover:text-foreground transition-colors"
                    >
                      <Linkedin className="h-6 w-6" />
                      <span className="sr-only">LinkedIn</span>
                    </a>
                    <a
                      href="mailto:nguyenphamquangdinh99@gmail.com"
                      className="text-muted-foreground hover:text-foreground transition-colors"
                    >
                      <Mail className="h-6 w-6" />
                      <span className="sr-only">Email</span>
                    </a>
                  </div>
                </div>
              </div>
            </div>

            <button
              onClick={() => scrollToSection('#about')}
              className="mx-auto mt-12 flex items-center justify-center text-muted-foreground hover:text-foreground transition-colors animate-bounce"
            >
              <ArrowDown className="h-6 w-6" />
              <span className="sr-only">Scroll to about section</span>
            </button>
          </div>
        </section>

        {/* About Section */}
        <section id="about" className="py-20 px-4 sm:px-6 lg:px-8">
          <div className="container mx-auto max-w-5xl">
            <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold mb-12 text-center text-balance">About Me</h2>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              {aboutCards.map((card, index) => {
                const Icon = card.icon;
                return (
                  <div
                    key={index}
                    className="glass rounded-2xl p-6 sm:p-8 hover:glass-strong transition-all duration-300 group"
                  >
                    <div className="flex items-start gap-4">
                      <div className="glass-subtle rounded-xl p-3 group-hover:bg-primary/10 transition-colors duration-300">
                        <Icon className="h-6 w-6 text-primary" />
                      </div>
                      <div className="flex-1">
                        <h3 className="text-xl font-semibold mb-3 text-balance">{card.title}</h3>
                        <p className="text-muted-foreground leading-relaxed text-pretty">{card.description}</p>
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        </section>

        {/* Skills Section */}
        <section id="skills" className="py-20 px-4 sm:px-6 lg:px-8">
          <div className="container mx-auto max-w-6xl">
            <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold mb-12 text-center text-balance">
              Skills & Expertise
            </h2>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              {skillCategories.map((category) => {
                const Icon = category.icon;
                return (
                  <div
                    key={category.title}
                    className="glass rounded-2xl p-6 sm:p-8 hover:glass-strong transition-all duration-300"
                  >
                    <div className="flex items-center gap-3 mb-6">
                      <div className="glass-subtle rounded-xl p-3">
                        <Icon className="h-6 w-6 text-primary" />
                      </div>
                      <h3 className="text-xl font-semibold">{category.title}</h3>
                    </div>

                    <div className="flex flex-wrap gap-2">
                      {category.skills.map((skill) => (
                        <span
                          key={skill}
                          className="glass-subtle rounded-lg px-3 py-1.5 text-sm font-medium text-foreground"
                        >
                          {skill}
                        </span>
                      ))}
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        </section>

        {/* Experience Section */}
        <section id="experience" className="py-20 px-4 sm:px-6 lg:px-8">
          <div className="container mx-auto max-w-5xl">
            <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold mb-12 text-center text-balance">
              Work Experience
            </h2>

            <div className="space-y-6">
              {experiences.map((exp, index) => (
                <div
                  key={index}
                  className="glass rounded-2xl p-6 sm:p-8 hover:glass-strong transition-all duration-300"
                >
                  <div className="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4 mb-4">
                    <div>
                      <h3 className="text-xl sm:text-2xl font-semibold mb-1">{exp.title}</h3>
                      <div className="flex items-center gap-2 text-primary">
                        <Briefcase className="h-4 w-4" />
                        <span className="font-medium">{exp.company}</span>
                      </div>
                    </div>
                    <div className="flex items-center gap-2 text-muted-foreground glass-subtle rounded-lg px-3 py-1.5 w-fit">
                      <Calendar className="h-4 w-4" />
                      <span className="text-sm font-medium">{exp.period}</span>
                    </div>
                  </div>

                  <p className="text-muted-foreground mb-4 leading-relaxed text-pretty">{exp.description}</p>

                  <div className="flex flex-wrap gap-2">
                    {exp.technologies.map((tech) => (
                      <span key={tech} className="glass-subtle rounded-lg px-3 py-1 text-sm font-medium text-primary">
                        {tech}
                      </span>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          </div>
        </section>

        {/* Projects Section */}
        <section id="projects" className="py-20 px-4 sm:px-6 lg:px-8">
          <div className="container mx-auto max-w-6xl">
            <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold mb-12 text-center text-balance">
              Featured Projects
            </h2>

            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              {projects.map((project, index) => (
                <div
                  key={index}
                  className="glass rounded-2xl overflow-hidden hover:glass-strong transition-all duration-300 group"
                >
                  <div className="relative h-48 sm:h-56 overflow-hidden">
                    <Image
                      src={project.image || '/placeholder.svg'}
                      alt={project.title}
                      fill
                      sizes="(min-width: 1024px) 50vw, 100vw"
                      className="object-cover transition-transform duration-300 group-hover:scale-105"
                      priority={index < 2}
                    />
                    <div className="absolute inset-0 bg-gradient-to-t from-background/80 to-transparent" />
                  </div>

                  <div className="p-6">
                    <h3 className="text-xl sm:text-2xl font-semibold mb-3">{project.title}</h3>

                    <p className="text-muted-foreground mb-4 leading-relaxed text-pretty">{project.description}</p>

                    <div className="flex flex-wrap gap-2 mb-6">
                      {project.technologies.map((tech) => (
                        <span key={tech} className="glass-subtle rounded-lg px-3 py-1 text-sm font-medium text-primary">
                          {tech}
                        </span>
                      ))}
                    </div>

                    <div className="flex gap-3">
                      <a
                        href={project.liveUrl}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="inline-flex items-center justify-center rounded-lg glass hover:glass-strong bg-transparent border border-border/50 px-4 py-2 text-sm font-medium transition-colors"
                      >
                        <ExternalLink className="mr-2 h-4 w-4" />
                        Live Demo
                      </a>
                      <a
                        href={project.githubUrl}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="inline-flex items-center justify-center rounded-lg glass hover:glass-strong bg-transparent border border-border/50 px-4 py-2 text-sm font-medium transition-colors"
                      >
                        <Github className="mr-2 h-4 w-4" />
                        Code
                      </a>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </section>

        {/* Hobbies Section */}
        <section id="hobbies" className="py-20 px-4 sm:px-6 lg:px-8">
          <div className="container mx-auto max-w-6xl">
            <div className="text-center mb-12">
              <h2 className="text-3xl sm:text-4xl font-bold text-foreground mb-4">Hobbies & Interests</h2>
              <p className="text-muted-foreground max-w-2xl mx-auto">
                Beyond coding, I enjoy a variety of activities that keep me balanced and inspired
              </p>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
              {hobbies.map((hobby, index) => {
                const Icon = hobby.icon;
                return (
                  <div
                    key={hobby.name}
                    className="glass-card p-6 rounded-2xl hover:scale-105 transition-all duration-300 group"
                    style={{
                      animationDelay: `${index * 100}ms`,
                    }}
                  >
                    <div className="flex flex-col items-center text-center space-y-4">
                      <div className="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center group-hover:bg-primary/20 transition-colors">
                        <Icon className="w-8 h-8 text-primary" />
                      </div>
                      <div>
                        <h3 className="text-lg font-semibold text-foreground mb-2">{hobby.name}</h3>
                        <p className="text-sm text-muted-foreground">{hobby.description}</p>
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        </section>

        {/* Contact Section */}
        <section id="contact" className="py-20 px-4 sm:px-6 lg:px-8">
          <div className="container mx-auto max-w-5xl">
            <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold mb-12 text-center text-balance">Get In Touch</h2>

            <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
              <div className="glass rounded-2xl p-8 space-y-8">
                <div>
                  <h3 className="text-2xl font-semibold mb-6">Let's work together</h3>
                  <p className="text-muted-foreground leading-relaxed text-pretty">
                    I'm always interested in hearing about new DevOps opportunities and cloud infrastructure projects.
                    Whether you have a question or just want to connect, feel free to reach out!
                  </p>
                </div>

                <div className="space-y-4">
                  <div className="flex items-start gap-4">
                    <div className="glass-subtle rounded-xl p-3">
                      <Mail className="h-5 w-5 text-primary" />
                    </div>
                    <div>
                      <p className="font-medium mb-1">Email</p>
                      <a
                        href="mailto:nguyenphamquangdinh99@gmail.com"
                        className="text-muted-foreground hover:text-foreground transition-colors"
                      >
                        nguyenphamquangdinh99@gmail.com
                      </a>
                    </div>
                  </div>

                  <div className="flex items-start gap-4">
                    <div className="glass-subtle rounded-xl p-3">
                      <Phone className="h-5 w-5 text-primary" />
                    </div>
                    <div>
                      <p className="font-medium mb-1">Phone</p>
                      <a
                        href="tel:+84366217657"
                        className="text-muted-foreground hover:text-foreground transition-colors"
                      >
                        +84 366 217 657
                      </a>
                    </div>
                  </div>

                  <div className="flex items-start gap-4">
                    <div className="glass-subtle rounded-xl p-3">
                      <MapPin className="h-5 w-5 text-primary" />
                    </div>
                    <div>
                      <p className="font-medium mb-1">Location</p>
                      <p className="text-muted-foreground">Ho Chi Minh City, Vietnam</p>
                    </div>
                  </div>
                </div>
              </div>

              <div className="glass rounded-2xl p-8">
                <form onSubmit={handleSubmit} className="space-y-6">
                  <div>
                    <label htmlFor="name" className="block text-sm font-medium mb-2">
                      Name
                    </label>
                    <input
                      id="name"
                      type="text"
                      placeholder="Your name"
                      value={formData.name}
                      onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                      className="w-full glass-subtle border border-border/50 rounded-lg px-4 py-2 text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/50 transition-all"
                      required
                    />
                  </div>

                  <div>
                    <label htmlFor="email" className="block text-sm font-medium mb-2">
                      Email
                    </label>
                    <input
                      id="email"
                      type="email"
                      placeholder="your.email@example.com"
                      value={formData.email}
                      onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                      className="w-full glass-subtle border border-border/50 rounded-lg px-4 py-2 text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/50 transition-all"
                      required
                    />
                  </div>

                  <div>
                    <label htmlFor="message" className="block text-sm font-medium mb-2">
                      Message
                    </label>
                    <textarea
                      id="message"
                      placeholder="Tell me about your project..."
                      value={formData.message}
                      onChange={(e) => setFormData({ ...formData, message: e.target.value })}
                      className="w-full glass-subtle border border-border/50 rounded-lg px-4 py-2 text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/50 transition-all min-h-32 resize-none"
                      required
                    />
                  </div>

                  <button
                    type="submit"
                    className="w-full inline-flex items-center justify-center rounded-lg bg-accent hover:bg-accent/90 text-white px-6 py-3 text-base font-medium transition-colors"
                  >
                    Send Message
                  </button>
                </form>
              </div>
            </div>

            <div className="mt-20 text-center">
              <div className="glass-subtle rounded-2xl p-6">
                <p className="text-sm text-muted-foreground">
                  © {new Date().getFullYear()} Quang Dinh Nguyen Pham. Built with Next.js and Tailwind CSS.
                </p>
              </div>
            </div>
          </div>
        </section>
      </div>
    </main>
  );
}
// export default Home;
