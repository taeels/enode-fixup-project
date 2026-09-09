# DDTHON 전체 캐릭터 시트

2026-09-09 · runixs · 사용자 요청으로 캐릭터 시트 제작을 재개한다.

원문: 영상을 만들껀데 ddthon-x.jpg 에서 캐릭터들을 뽑아서 캐릭터시트를 캐릭터별로 전부 만들어놔줘.

ddthon-x라는 실제 파일은 없으며 루트의 ddthon-1.jpg, ddthon-2.png,
ddthon-3.jpeg, ddthon-4.jpeg, ddthon-5.jpeg 전체를 시리즈로 해석한다.
동일한 마스코트의 반복 등장과 같은 로봇 말의 복제는 한 종류로 묶어 총 9종이다.
아래 명칭은 작업용 이름이며 공식 캐릭터 이름으로 주장하지 않는다.

출력: 전체 시트 폴더 (`output/enode-video/character-sheets/ddthon-all-v1/`).
생성 도구: 내장 image_gen. 전체 프롬프트 (`output/enode-video/character-sheets/ddthon-all-v1/prompts.json`).

## 구성과 보완 범위

캐릭터마다 정면·3/4·측면·후면 전신, 기본·기쁨·놀람·집중 표정과 색상 견본을
한 PNG에 구성한다. 원본 2D 평면 일러스트를 기준으로 한다. 사진 자료의 조명·인쇄색
변화는 디지털 원본을 우선해 보정한다. 가려진 하체·후면·측면은 생성으로 보완한
영상용 디자인이며 공식 원화나 픽셀 그대로의 누끼 추출이 아니다.

## 진행

- [x] 원본 5개와 기존 영상·캐릭터 보류 기록 확인.
- [x] 중복을 합친 9종 목록·고정 특징·개별 프롬프트 확정.
- [x] 01 민트 학자: 생성·검수·저장.
- [x] 02 파란 기사: 생성·검수·저장.
- [x] 03 하늘색 보조 기사: 생성·검수·저장.
- [x] 04 빨간 별 검객: 생성·검수·저장.
- [x] 05 보라 궤도 행성: 생성·검수·저장.
- [x] 06 금색 전차 기사: 생성·검수·저장.
- [x] 07 주황 뿔 전차 기사: 생성·검수·저장.
- [x] 08 흰 로봇 말: 생성·검수·저장.
- [x] 09 빨간 로봇 말: 생성·검수·저장.
- [x] 파일 무결성·목록·일괄 다운로드 묶음 확인.

기존 캐릭터 초안과 채택 영상은 별도로 보존한다. 이번 요청 범위는 참조 이미지
제작이며 소프트웨어 개발 단계나 장면 게이트 완료로 기록하지 않는다.

## 완성 파일

모든 PNG는 1536 × 1024, 흰 배경이다. 아래 이름은 작업용 구분명이다.

| 번호 | 캐릭터 시트 | 원본 | 고정 특징 |
|---|---|---|---|
| 1 | 민트 학자 (`output/enode-video/character-sheets/ddthon-all-v1/01-mint-scholar.png`) | ddthon-1.jpg, ddthon-3.jpeg | 민트 몸·분화구 무늬·둥근 안경·구름 머리·흰 띠 |
| 2 | 파란 기사 (`output/enode-video/character-sheets/ddthon-all-v1/02-blue-knight.png`) | ddthon-1.jpg, ddthon-4.jpeg, ddthon-3.jpeg | 파란 구름형 몸·회색 뿔 투구·갈색 대각선 끈·창 |
| 3 | 하늘색 보조 기사 (`output/enode-video/character-sheets/ddthon-all-v1/03-cyan-squire.png`) | ddthon-1.jpg, ddthon-4.jpeg, ddthon-3.jpeg | 하늘색 몸·곡선 장식 투구·흰 띠 |
| 4 | 빨간 별 검객 (`output/enode-video/character-sheets/ddthon-all-v1/04-red-star-swordsman.png`) | ddthon-5.jpeg, ddthon-4.jpeg, ddthon-1.jpg | 별 모양 눈동자·금색 허리 고리·파란 망토·검 |
| 5 | 보라 궤도 행성 (`output/enode-video/character-sheets/ddthon-all-v1/05-lavender-orbit.png`) | ddthon-5.jpeg, ddthon-1.jpg | 보라 구형 몸·궤도 장식·이마 반짝임·금색 메달 |
| 6 | 금색 전차 기사 (`output/enode-video/character-sheets/ddthon-all-v1/06-gold-charioteer.png`) | ddthon-2.png | 금색 투구·붉은 볏·파란 망토·AI-DLC 표기 |
| 7 | 주황 뿔 전차 기사 (`output/enode-video/character-sheets/ddthon-all-v1/07-orange-horned-charioteer.png`) | ddthon-2.png | 주황 몸·회색 뿔 투구·작은 불꽃 장식·붉은 망토 |
| 8 | 흰 로봇 말 (`output/enode-video/character-sheets/ddthon-all-v1/08-white-robot-horse.png`) | ddthon-2.png | 네 발·흰 몸·검은 바이저·별 반사·청록 귀 고리 |
| 9 | 빨간 로봇 말 (`output/enode-video/character-sheets/ddthon-all-v1/09-red-robot-horse.png`) | ddthon-2.png | 네 발·빨간 몸·검은 바이저·붉은 눈 표시·기계 귀 |

전체 ZIP (`output/enode-video/character-sheets/ddthon-all-v1.zip`)에는 PNG 9개, 프롬프트와 무결성 manifest를 넣었다.

## 검수와 영상 참조

9장의 색상·외형·복장·장비·중복 캐릭터 혼입·잘림을 원본과 대조했다.
두 로봇 말은 네 발과 바이저를 유지한다. 정면·3/4·측면·후면은 생성 모델의
턴어라운드 해석이며 기계적으로 회전시킨 정투영 모델은 아니다. 원본에 없는
표정과 가려진 부분은 새로 보완했다. 파일의 PNG 청크 CRC·압축 데이터·크기·
SHA-256 및 ZIP 무결성을 검사했다.

향후 영상 제작에서는 해당 장면에 등장하는 캐릭터의 시트를 고정 참조로 사용한다.
시트 전체를 장면 배경으로 재현하지 않도록 실제 장면 프레임과 구분한다. 기존
영상의 파란 요청자는 02, 금색 동료는 06, 흰 로봇 말은 08에 대응한다.
서대현님의 휴대폰·손은 DDTHON 원본 등장 캐릭터가 아니어서 이번 9종에 추가하지 않았다.

## 로컬 미리보기

위 표의 PNG는 로컬 제작 폴더에 보존한다. 이미지 자체는 Git에 포함하지 않는다.
