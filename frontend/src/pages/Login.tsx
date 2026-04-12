import React, { useState, useEffect, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { Form, Input, Button, Typography, message } from 'antd';
import { UserOutlined, LockOutlined, LoginOutlined } from '@ant-design/icons';
import { useAuth } from '../contexts/AuthContext';

const { Title, Text } = Typography;

// --- 1. Subcomponents for Eyes and Pupils ---

interface PupilProps {
  size?: number;
  maxDistance?: number;
  pupilColor?: string;
  forceLookX?: number;
  forceLookY?: number;
  isClosed?: boolean;
}

const Pupil = ({
  size = 12,
  maxDistance = 5,
  pupilColor = 'black',
  forceLookX,
  forceLookY,
  isClosed = false,
}: PupilProps) => {
  const [mouseX, setMouseX] = useState<number>(0);
  const [mouseY, setMouseY] = useState<number>(0);
  const pupilRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      setMouseX(e.clientX);
      setMouseY(e.clientY);
    };
    window.addEventListener('mousemove', handleMouseMove);
    return () => window.removeEventListener('mousemove', handleMouseMove);
  }, []);

  const calculatePupilPosition = () => {
    if (!pupilRef.current) return { x: 0, y: 0 };
    if (forceLookX !== undefined && forceLookY !== undefined) {
      return { x: forceLookX, y: forceLookY };
    }
    const pupil = pupilRef.current.getBoundingClientRect();
    const pupilCenterX = pupil.left + pupil.width / 2;
    const pupilCenterY = pupil.top + pupil.height / 2;

    const deltaX = mouseX - pupilCenterX;
    const deltaY = mouseY - pupilCenterY;
    const distance = Math.min(
      Math.sqrt(deltaX ** 2 + deltaY ** 2),
      maxDistance,
    );

    const angle = Math.atan2(deltaY, deltaX);
    const x = Math.cos(angle) * distance;
    const y = Math.sin(angle) * distance;
    return { x, y };
  };

  const pupilPosition = calculatePupilPosition();

  return (
    <div
      ref={pupilRef}
      style={{
        borderRadius: isClosed ? '2px' : '50%',
        width: `${size}px`,
        height: isClosed ? '2px' : `${size}px`,
        backgroundColor: pupilColor,
        transform: `translate(${pupilPosition.x}px, ${isClosed ? 0 : pupilPosition.y}px)`,
        transition: 'all 0.15s ease-out',
      }}
    />
  );
};

interface EyeBallProps {
  size?: number;
  pupilSize?: number;
  maxDistance?: number;
  eyeColor?: string;
  pupilColor?: string;
  isBlinking?: boolean;
  forceLookX?: number;
  forceLookY?: number;
  isClosed?: boolean;
}

const EyeBall = ({
  size = 48,
  pupilSize = 16,
  maxDistance = 10,
  eyeColor = 'white',
  pupilColor = 'black',
  isBlinking = false,
  forceLookX,
  forceLookY,
  isClosed = false,
}: EyeBallProps) => {
  const [mouseX, setMouseX] = useState<number>(0);
  const [mouseY, setMouseY] = useState<number>(0);
  const eyeRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      setMouseX(e.clientX);
      setMouseY(e.clientY);
    };
    window.addEventListener('mousemove', handleMouseMove);
    return () => window.removeEventListener('mousemove', handleMouseMove);
  }, []);

  const calculatePupilPosition = () => {
    if (!eyeRef.current) return { x: 0, y: 0 };
    if (forceLookX !== undefined && forceLookY !== undefined) {
      return { x: forceLookX, y: forceLookY };
    }
    const eye = eyeRef.current.getBoundingClientRect();
    const eyeCenterX = eye.left + eye.width / 2;
    const eyeCenterY = eye.top + eye.height / 2;

    const deltaX = mouseX - eyeCenterX;
    const deltaY = mouseY - eyeCenterY;
    const distance = Math.min(
      Math.sqrt(deltaX ** 2 + deltaY ** 2),
      maxDistance,
    );

    const angle = Math.atan2(deltaY, deltaX);
    const x = Math.cos(angle) * distance;
    const y = Math.sin(angle) * distance;
    return { x, y };
  };

  const pupilPosition = calculatePupilPosition();
  const closed = isBlinking || isClosed;

  return (
    <div
      ref={eyeRef}
      style={{
        borderRadius: '50%',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        transition: 'all 0.15s ease',
        width: `${size}px`,
        height: closed ? '2px' : `${size}px`,
        backgroundColor: eyeColor,
        overflow: 'hidden',
        boxShadow:
          eyeColor === '#3A3A3A' ? 'inset 0 0 4px rgba(0,0,0,0.5)' : undefined,
      }}
    >
      {!closed && (
        <div
          style={{
            borderRadius: '50%',
            width: `${pupilSize}px`,
            height: `${pupilSize}px`,
            backgroundColor: pupilColor,
            transform: `translate(${pupilPosition.x}px, ${pupilPosition.y}px)`,
            transition: 'transform 0.1s ease-out',
          }}
        />
      )}
    </div>
  );
};

// --- 2. Main Login Component ---

const Login = () => {
  const [loading, setLoading] = useState(false);
  const [form] = Form.useForm();
  const navigate = useNavigate();
  const { login } = useAuth();
  const [isTransitioning, setIsTransitioning] = useState(false);

  // Character States
  const [passwordVisible, setPasswordVisible] = useState(false);
  const [isTyping, setIsTyping] = useState(false);
  const [isPasswordFocused, setIsPasswordFocused] = useState(false);
  const [mouseX, setMouseX] = useState<number>(0);
  const [mouseY, setMouseY] = useState<number>(0);

  const [isPurpleBlinking, setIsPurpleBlinking] = useState(false);
  const [isBlackBlinking, setIsBlackBlinking] = useState(false);
  const [isLookingAtEachOther, setIsLookingAtEachOther] = useState(false);
  const [isPurplePeeking, setIsPurplePeeking] = useState(false);

  // Refs for characters
  const purpleRef = useRef<HTMLDivElement>(null);
  const blackRef = useRef<HTMLDivElement>(null);
  const yellowRef = useRef<HTMLDivElement>(null);
  const orangeRef = useRef<HTMLDivElement>(null);

  // Mouse move handler
  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      setMouseX(e.clientX);
      setMouseY(e.clientY);
    };
    window.addEventListener('mousemove', handleMouseMove);
    return () => window.removeEventListener('mousemove', handleMouseMove);
  }, []);

  // Blinking effects
  useEffect(() => {
    const getRandomBlinkInterval = () => Math.random() * 6000 + 5000;
    const scheduleBlink = () => {
      return setTimeout(() => {
        setIsPurpleBlinking(true);
        setTimeout(() => {
          setIsPurpleBlinking(false);
          scheduleBlink();
        }, 150);
      }, getRandomBlinkInterval());
    };
    const timeout = scheduleBlink();
    return () => clearTimeout(timeout);
  }, []);

  useEffect(() => {
    const getRandomBlinkInterval = () => Math.random() * 6000 + 5000;
    const scheduleBlink = () => {
      return setTimeout(() => {
        setIsBlackBlinking(true);
        setTimeout(() => {
          setIsBlackBlinking(false);
          scheduleBlink();
        }, 150);
      }, getRandomBlinkInterval());
    };
    const timeout = scheduleBlink();
    return () => clearTimeout(timeout);
  }, []);

  // Looking at each other when typing in ID field
  useEffect(() => {
    if (isTyping) {
      setIsLookingAtEachOther(true);
      const timer = setTimeout(() => {
        setIsLookingAtEachOther(false);
      }, 800);
      return () => clearTimeout(timer);
    } else {
      setIsLookingAtEachOther(false);
    }
  }, [isTyping]);

  // Purple randomly peeking when avoiding looking at the password
  useEffect(() => {
    let timeoutId: any;
    let peekTimeoutId: any;
    let isActive = true;

    if (isPasswordFocused && !passwordVisible) {
      const schedulePeek = () => {
        timeoutId = setTimeout(
          () => {
            if (!isActive) return;
            setIsPurplePeeking(true);
            peekTimeoutId = setTimeout(
              () => {
                if (!isActive) return;
                setIsPurplePeeking(false);
                schedulePeek();
              },
              500 + Math.random() * 300,
            );
          },
          Math.random() * 3000 + 2000,
        );
      };
      schedulePeek();
    } else {
      setIsPurplePeeking(false);
    }

    return () => {
      isActive = false;
      clearTimeout(timeoutId);
      clearTimeout(peekTimeoutId);
    };
  }, [isPasswordFocused, passwordVisible]);

  const calculatePosition = (ref: React.RefObject<HTMLDivElement | null>) => {
    if (!ref.current) return { faceX: 0, faceY: 0, bodySkew: 0 };
    const rect = ref.current.getBoundingClientRect();
    const centerX = rect.left + rect.width / 2;
    const centerY = rect.top + rect.height / 3;
    const deltaX = mouseX - centerX;
    const deltaY = mouseY - centerY;

    const faceX = Math.max(-15, Math.min(15, deltaX / 20));
    const faceY = Math.max(-10, Math.min(10, deltaY / 30));
    const bodySkew = Math.max(-6, Math.min(6, -deltaX / 120));
    return { faceX, faceY, bodySkew };
  };

  const purplePos = calculatePosition(purpleRef);
  const blackPos = calculatePosition(blackRef);
  const yellowPos = calculatePosition(yellowRef);
  const orangePos = calculatePosition(orangeRef);

  const shouldHide = isPasswordFocused && !passwordVisible;
  const shouldPeek = isPasswordFocused && passwordVisible;

  // Form Submit
  const handleSubmit = async (values: {
    employee_id: string;
    password: string;
  }) => {
    setLoading(true);
    try {
      await login(values.employee_id, values.password);
      message.success('登录成功');
      // 启动过渡动画 - 使用更流畅的动画序列
      setIsTransitioning(true);
      // 延迟导航，让动画播放更自然
      setTimeout(() => {
        navigate('/dashboard');
      }, 800);
    } catch (error: any) {
      console.error('Login failed:', error);
      const serverMsg = error?.response?.data?.error;
      message.error(serverMsg || '登录失败，请检查工号和密码');
    } finally {
      if (!isTransitioning) {
        setLoading(false);
      }
    }
  };

  return (
    <>
      <style>{`
        @keyframes fadeInUp {
          from {
            opacity: 0;
            transform: translateY(20px);
          }
          to {
            opacity: 1;
            transform: translateY(0);
          }
        }

        @keyframes fadeIn {
          from {
            opacity: 0;
          }
          to {
            opacity: 1;
          }
        }

        @keyframes scaleIn {
          from {
            opacity: 0;
            transform: scale(0.9);
          }
          to {
            opacity: 1;
            transform: scale(1);
          }
        }

        @keyframes slideInRight {
          from {
            opacity: 0;
            transform: translateX(20px);
          }
          to {
            opacity: 1;
            transform: translateX(0);
          }
        }

        @keyframes fadeOutLeft {
          from {
            opacity: 1;
            transform: translateX(0);
          }
          to {
            opacity: 0;
            transform: translateX(-30px);
          }
        }

        @keyframes scaleOut {
          from {
            opacity: 1;
            transform: scale(1);
          }
          to {
            opacity: 0;
            transform: scale(0.95);
          }
        }

        .login-wrapper {
          display: flex;
          min-height: 100vh;
          width: 100vw;
          overflow: hidden;
          background-color: #ffffff;
          transition: all 0.8s cubic-bezier(0.4, 0, 0.2, 1);
        }

        .login-wrapper.transitioning {
          animation: scaleOut 0.8s cubic-bezier(0.4, 0, 0.2, 1) forwards;
        }

        .login-left {
          flex: 1;
          display: flex;
          flex-direction: column;
          justify-content: space-between;
          background: linear-gradient(135deg, #1e1b4b 0%, #312e81 100%);
          padding: 48px;
          color: white;
          position: relative;
          overflow: hidden;
          animation: fadeIn 0.6s ease-out;
        }

        .login-left.transitioning {
          animation: fadeOutLeft 0.8s cubic-bezier(0.4, 0, 0.2, 1) forwards;
        }

        .login-right {
          flex: 1;
          display: flex;
          align-items: center;
          justify-content: center;
          padding: 32px;
          background-color: #ffffff;
          position: relative;
          z-index: 10;
          box-shadow: -10px 0 30px rgba(0,0,0,0.05);
          animation: fadeIn 0.6s ease-out;
        }

        .login-logo {
          animation: scaleIn 0.4s cubic-bezier(0.4, 0, 0.2, 1) 0.1s both;
        }

        .login-title {
          animation: fadeInUp 0.4s cubic-bezier(0.4, 0, 0.2, 1) 0.2s both;
        }

        .login-subtitle {
          animation: fadeInUp 0.4s cubic-bezier(0.4, 0, 0.2, 1) 0.3s both;
        }

        .login-form {
          animation: fadeInUp 0.5s cubic-bezier(0.4, 0, 0.2, 1) 0.4s both;
        }

        .login-form-item {
          animation: slideInRight 0.3s cubic-bezier(0.4, 0, 0.2, 1) both;
        }

        .login-form-item:nth-child(1) {
          animation-delay: 0.5s;
        }

        .login-form-item:nth-child(2) {
          animation-delay: 0.6s;
        }

        .login-form-item:nth-child(3) {
          animation-delay: 0.7s;
        }

        .login-input {
          transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
        }

        .login-input:hover {
          border-color: #4f46e5 !important;
          box-shadow: 0 0 0 2px rgba(79, 70, 229, 0.1);
        }

        .login-input:focus,
        .login-input.ant-input-focused {
          border-color: #4f46e5 !important;
          box-shadow: 0 0 0 3px rgba(79, 70, 229, 0.2);
        }

        .login-button {
          transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
        }

        .login-button:hover {
          transform: translateY(-1px);
          box-shadow: 0 4px 12px rgba(79, 70, 229, 0.4) !important;
        }

        .login-button:active {
          transform: translateY(0) scale(0.98);
        }

        .login-error {
          animation: fadeInUp 0.2s ease-out;
        }

        @media (max-width: 1024px) {
          .login-left {
            display: none;
          }
        }
      `}</style>

      <div className={`login-wrapper ${isTransitioning ? 'transitioning' : ''}`}>
        {/* Left Content Section */}
        <div className={`login-left ${isTransitioning ? 'transitioning' : ''}`}>
          {/* Decorative background effects */}
          <div
            style={{
              position: 'absolute',
              top: '25%',
              right: '25%',
              width: '250px',
              height: '250px',
              backgroundColor: 'rgba(255,255,255,0.05)',
              borderRadius: '50%',
              filter: 'blur(60px)',
              zIndex: 1,
            }}
          />
          <div
            style={{
              position: 'absolute',
              bottom: '25%',
              left: '25%',
              width: '350px',
              height: '350px',
              backgroundColor: 'rgba(255,255,255,0.02)',
              borderRadius: '50%',
              filter: 'blur(80px)',
              zIndex: 1,
            }}
          />

          <div
            style={{
              position: 'relative',
              zIndex: 20,
              display: 'flex',
              alignItems: 'flex-end',
              justifyContent: 'center',
              height: '500px',
            }}
          >
            {/* Cartoon Characters Container */}
            <div
              style={{ position: 'relative', width: '550px', height: '400px' }}
            >
              {/* Purple tall rectangle character */}
              <div
                ref={purpleRef}
                style={{
                  position: 'absolute',
                  bottom: 0,
                  left: '70px',
                  width: '180px',
                  height: isTyping || shouldHide ? '440px' : '400px',
                  backgroundColor: '#6C3FF5',
                  borderRadius: '10px 10px 0 0',
                  zIndex: 1,
                  transition: 'all 0.7s ease-in-out',
                  transformOrigin: 'bottom center',
                  transform: shouldPeek
                    ? `skewX(0deg)`
                    : shouldHide
                      ? `skewX(12deg) translateX(-30px)`
                      : isTyping
                        ? `skewX(${(purplePos.bodySkew || 0) - 12}deg) translateX(40px)`
                        : `skewX(${purplePos.bodySkew || 0}deg)`,
                }}
              >
                {/* Eyes */}
                <div
                  style={{
                    position: 'absolute',
                    display: 'flex',
                    gap: '32px',
                    transition: 'all 0.7s ease-in-out',
                    left: shouldPeek
                      ? '20px'
                      : isLookingAtEachOther
                        ? '55px'
                        : `${45 + purplePos.faceX}px`,
                    top: shouldPeek
                      ? '35px'
                      : isLookingAtEachOther
                        ? '65px'
                        : `${40 + purplePos.faceY}px`,
                  }}
                >
                  <EyeBall
                    size={18}
                    pupilSize={7}
                    maxDistance={5}
                    eyeColor="white"
                    pupilColor="#2D2D2D"
                    isBlinking={isPurpleBlinking}
                    forceLookX={
                      shouldPeek
                        ? undefined
                        : shouldHide
                          ? isPurplePeeking
                            ? 5
                            : -5
                          : isLookingAtEachOther
                            ? 3
                            : undefined
                    }
                    forceLookY={
                      shouldPeek
                        ? undefined
                        : shouldHide
                          ? isPurplePeeking
                            ? 0
                            : 2
                          : isLookingAtEachOther
                            ? 4
                            : undefined
                    }
                  />
                  <EyeBall
                    size={18}
                    pupilSize={7}
                    maxDistance={5}
                    eyeColor="white"
                    pupilColor="#2D2D2D"
                    isBlinking={isPurpleBlinking}
                    forceLookX={
                      shouldPeek
                        ? undefined
                        : shouldHide
                          ? isPurplePeeking
                            ? 5
                            : -5
                          : isLookingAtEachOther
                            ? 3
                            : undefined
                    }
                    forceLookY={
                      shouldPeek
                        ? undefined
                        : shouldHide
                          ? isPurplePeeking
                            ? 0
                            : 2
                          : isLookingAtEachOther
                            ? 4
                            : undefined
                    }
                  />
                </div>
              </div>

              {/* Black tall rectangle character */}
              <div
                ref={blackRef}
                style={{
                  position: 'absolute',
                  bottom: 0,
                  left: '240px',
                  width: '120px',
                  height: '310px',
                  backgroundColor: '#2D2D2D',
                  borderRadius: '8px 8px 0 0',
                  zIndex: 2,
                  transition: 'all 0.7s ease-in-out',
                  transformOrigin: 'bottom center',
                  transform: shouldPeek
                    ? `skewX(0deg)`
                    : shouldHide
                      ? `skewX(12deg) translateX(-30px)`
                      : isLookingAtEachOther
                        ? `skewX(${(blackPos.bodySkew || 0) * 1.5 + 10}deg) translateX(20px)`
                        : isTyping
                          ? `skewX(${(blackPos.bodySkew || 0) * 1.5}deg)`
                          : `skewX(${blackPos.bodySkew || 0}deg)`,
                }}
              >
                {/* Eyes */}
                <div
                  style={{
                    position: 'absolute',
                    display: 'flex',
                    gap: '24px',
                    transition: 'all 0.7s ease-in-out',
                    left: shouldPeek
                      ? '10px'
                      : isLookingAtEachOther
                        ? '32px'
                        : `${26 + blackPos.faceX}px`,
                    top: shouldPeek
                      ? '28px'
                      : isLookingAtEachOther
                        ? '12px'
                        : `${32 + blackPos.faceY}px`,
                  }}
                >
                  <EyeBall
                    size={16}
                    pupilSize={6}
                    maxDistance={4}
                    eyeColor="white"
                    pupilColor="#2D2D2D"
                    isBlinking={isBlackBlinking}
                    forceLookX={
                      shouldPeek
                        ? undefined
                        : shouldHide
                          ? -4
                          : isLookingAtEachOther
                            ? 0
                            : undefined
                    }
                    forceLookY={
                      shouldPeek
                        ? undefined
                        : shouldHide
                          ? 2
                          : isLookingAtEachOther
                            ? -4
                            : undefined
                    }
                  />
                  <EyeBall
                    size={16}
                    pupilSize={6}
                    maxDistance={4}
                    eyeColor="white"
                    pupilColor="#2D2D2D"
                    isBlinking={isBlackBlinking}
                    forceLookX={
                      shouldPeek
                        ? undefined
                        : shouldHide
                          ? -4
                          : isLookingAtEachOther
                            ? 0
                            : undefined
                    }
                    forceLookY={
                      shouldPeek
                        ? undefined
                        : shouldHide
                          ? 2
                          : isLookingAtEachOther
                            ? -4
                            : undefined
                    }
                  />
                </div>
              </div>

              {/* Orange semi-circle character */}
              <div
                ref={orangeRef}
                style={{
                  position: 'absolute',
                  bottom: 0,
                  left: '0px',
                  width: '240px',
                  height: '200px',
                  zIndex: 3,
                  backgroundColor: '#FF9B6B',
                  borderRadius: '120px 120px 0 0',
                  transition: 'all 0.7s ease-in-out',
                  transformOrigin: 'bottom center',
                  transform: shouldPeek
                    ? `skewX(0deg)`
                    : shouldHide
                      ? `skewX(10deg) translateX(-20px)`
                      : `skewX(${orangePos.bodySkew || 0}deg)`,
                }}
              >
                {/* Eyes */}
                <div
                  style={{
                    position: 'absolute',
                    display: 'flex',
                    gap: '32px',
                    transition: 'all 0.2s ease-out',
                    left: shouldPeek
                      ? '50px'
                      : `${82 + (orangePos.faceX || 0)}px`,
                    top: shouldPeek
                      ? '85px'
                      : `${90 + (orangePos.faceY || 0)}px`,
                  }}
                >
                  <Pupil
                    size={12}
                    maxDistance={5}
                    pupilColor="#2D2D2D"
                    forceLookX={
                      shouldPeek ? undefined : shouldHide ? -4 : undefined
                    }
                    forceLookY={
                      shouldPeek ? undefined : shouldHide ? 2 : undefined
                    }
                  />
                  <Pupil
                    size={12}
                    maxDistance={5}
                    pupilColor="#2D2D2D"
                    forceLookX={
                      shouldPeek ? undefined : shouldHide ? -4 : undefined
                    }
                    forceLookY={
                      shouldPeek ? undefined : shouldHide ? 2 : undefined
                    }
                  />
                </div>
              </div>

              {/* Yellow tall rectangle character */}
              <div
                ref={yellowRef}
                style={{
                  position: 'absolute',
                  bottom: 0,
                  left: '310px',
                  width: '140px',
                  height: '230px',
                  backgroundColor: '#E8D754',
                  borderRadius: '70px 70px 0 0',
                  zIndex: 4,
                  transition: 'all 0.7s ease-in-out',
                  transformOrigin: 'bottom center',
                  transform: shouldPeek
                    ? `skewX(0deg)`
                    : shouldHide
                      ? `skewX(12deg) translateX(-30px)`
                      : `skewX(${yellowPos.bodySkew || 0}deg)`,
                }}
              >
                {/* Eyes */}
                <div
                  style={{
                    position: 'absolute',
                    display: 'flex',
                    gap: '24px',
                    transition: 'all 0.2s ease-out',
                    left: shouldPeek
                      ? '20px'
                      : `${52 + (yellowPos.faceX || 0)}px`,
                    top: shouldPeek
                      ? '35px'
                      : `${40 + (yellowPos.faceY || 0)}px`,
                  }}
                >
                  <Pupil
                    size={12}
                    maxDistance={5}
                    pupilColor="#2D2D2D"
                    forceLookX={
                      shouldPeek ? undefined : shouldHide ? -4 : undefined
                    }
                    forceLookY={
                      shouldPeek ? undefined : shouldHide ? 2 : undefined
                    }
                  />
                  <Pupil
                    size={12}
                    maxDistance={5}
                    pupilColor="#2D2D2D"
                    forceLookX={
                      shouldPeek ? undefined : shouldHide ? -4 : undefined
                    }
                    forceLookY={
                      shouldPeek ? undefined : shouldHide ? 2 : undefined
                    }
                  />
                </div>
                {/* Mouth */}
                <div
                  style={{
                    position: 'absolute',
                    width: '80px',
                    height: '4px',
                    backgroundColor: '#2D2D2D',
                    borderRadius: '2px',
                    transition: 'all 0.2s ease-out',
                    left: shouldPeek
                      ? '10px'
                      : `${40 + (yellowPos.faceX || 0)}px`,
                    top: shouldPeek
                      ? '88px'
                      : `${88 + (yellowPos.faceY || 0)}px`,
                  }}
                />
              </div>
            </div>
          </div>

          <div
            style={{
              position: 'relative',
              zIndex: 20,
              display: 'flex',
              alignItems: 'center',
              gap: '32px',
              fontSize: '14px',
              color: 'rgba(255,255,255,0.6)',
            }}
          ></div>
        </div>

        {/* Right Side: Login Form */}
        <div className="login-right">
          <div style={{ width: '100%', maxWidth: '380px' }}>
            <div style={{ textAlign: 'center', marginBottom: '40px' }}>
              <div
                className="login-logo"
                style={{
                  width: '64px',
                  height: '64px',
                  background:
                    'linear-gradient(135deg, #1e1b4b 0%, #4f46e5 100%)',
                  borderRadius: '16px',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  margin: '0 auto 16px',
                  boxShadow: '0 8px 16px rgba(79, 70, 229, 0.2)',
                }}
              >
                <LoginOutlined style={{ fontSize: '32px', color: 'white' }} />
              </div>
              <Title
                level={2}
                className="login-title"
                style={{
                  margin: '0 0 8px 0',
                  color: '#1a202c',
                  fontWeight: 700,
                }}
              >
                欢迎回来
              </Title>
              <Text type="secondary" className="login-subtitle" style={{ fontSize: '15px' }}>
                离线编程程序管理系统
              </Text>
            </div>

            <Form
              form={form}
              name="login"
              onFinish={handleSubmit}
              layout="vertical"
              size="large"
              className="login-form"
            >
              <Form.Item
                name="employee_id"
                rules={[{ required: true, message: '请输入工号' }]}
                className="login-form-item"
              >
                <Input
                  prefix={
                    <UserOutlined
                      style={{ color: '#a0aec0', marginRight: '8px' }}
                    />
                  }
                  placeholder="请输入工号"
                  onFocus={() => setIsTyping(true)}
                  onBlur={() => setIsTyping(false)}
                  className="login-input"
                  style={{
                    borderRadius: '12px',
                    padding: '12px 16px',
                    backgroundColor: '#f8fafc',
                    border: '1px solid #e2e8f0',
                  }}
                />
              </Form.Item>

              <Form.Item
                name="password"
                rules={[{ required: true, message: '请输入密码' }]}
                style={{ marginBottom: '24px' }}
                className="login-form-item"
              >
                <Input.Password
                  prefix={
                    <LockOutlined
                      style={{ color: '#a0aec0', marginRight: '8px' }}
                    />
                  }
                  placeholder="请输入密码"
                  onFocus={() => setIsPasswordFocused(true)}
                  onBlur={() => setIsPasswordFocused(false)}
                  visibilityToggle={{
                    visible: passwordVisible,
                    onVisibleChange: setPasswordVisible,
                  }}
                  className="login-input"
                  style={{
                    borderRadius: '12px',
                    padding: '12px 16px',
                    backgroundColor: '#f8fafc',
                    border: '1px solid #e2e8f0',
                  }}
                />
              </Form.Item>

              <Form.Item style={{ marginBottom: '16px' }} className="login-form-item">
                <Button
                  type="primary"
                  htmlType="submit"
                  loading={loading}
                  block
                  className="login-button"
                  style={{
                    height: '52px',
                    background:
                      'linear-gradient(135deg, #4f46e5 0%, #3730a3 100%)',
                    border: 'none',
                    borderRadius: '12px',
                    fontSize: '16px',
                    fontWeight: '600',
                    boxShadow: '0 4px 14px rgba(79, 70, 229, 0.3)',
                  }}
                >
                  {loading ? '登录中...' : '登录'}
                </Button>
              </Form.Item>
            </Form>

            <div style={{ textAlign: 'center', marginTop: '24px' }} className="login-title">
              <Text type="secondary" style={{ fontSize: '13px' }}>
                忘记密码？请联系系统管理员
              </Text>
            </div>
          </div>
        </div>
      </div>
    </>
  );
};

export default Login;
