<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<ul id="paramNodesUl" class="paramNodesUl">
<%-- DRX开关 --%>
	<li>
		<label for="LTE_DRX_ENABLE_name">${LTE_DRX_ENABLE_name }</label>
		<select id="LTE_DRX_ENABLE_name" name="LTE_DRX_ENABLE" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="1">true</option>
			<option value="0">false</option>
		</select>
	</li>
	<%-- DRX不活动定时器 --%>
	<li>
		<label for="LTE_DRX_INACTIVITY_TIMER_name">${LTE_DRX_INACTIVITY_TIMER_name }</label>
		<select id="LTE_DRX_INACTIVITY_TIMER_name" name="LTE_DRX_INACTIVITY_TIMER" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="1">1</option>
			<option value="2">2</option>
			<option value="3">3</option>
			<option value="4">4</option>
			<option value="5">5</option>
			<option value="6">6</option>
			<option value="8">8</option>
			<option value="10">10</option>
			<option value="20">20</option>
			<option value="30">30</option>
			<option value="40">40</option>
			<option value="50">50</option>
			<option value="60">60</option>
			<option value="80">80</option>
			<option value="100">100</option>
			<option value="200">200</option>
			<option value="300">300</option>
			<option value="500">500</option>
			<option value="750">750</option>
			<option value="1280">1280</option>
			<option value="1920">1920</option>
			<option value="2560">2560</option>
		</select>
	</li>
	<%-- LongDRXCycleGBR --%>
	<li>
		<label for="LTE_LONG_DRX_CYCLE_GBR_name">${LTE_LONG_DRX_CYCLE_GBR_name }</label>
		<select id="LTE_LONG_DRX_CYCLE_GBR_name" name="LTE_LONG_DRX_CYCLE_GBR" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="10">10</option>
			<option value="20">20</option>
			<option value="32">32</option>
			<option value="40">40</option>
			<option value="64">64</option>
			<option value="80">80</option>
			<option value="128">128</option>
			<option value="160">160</option>
			<option value="256">256</option>
			<option value="320">320</option>
			<option value="512">512</option>
			<option value="640">640</option>
			<option value="1024">1024</option>
			<option value="1280">1280</option>
			<option value="2048">2048</option>
			<option value="2560">2560</option>
		</select>
	</li>
	<%-- LongDRXCycleNonGBR --%>
	<li>
		<label for="LTE_LONG_DRX_CYCLE_NON_GBR_name">${LTE_LONG_DRX_CYCLE_NON_GBR_name }</label>
		<select id="LTE_LONG_DRX_CYCLE_NON_GBR_name" name="LTE_LONG_DRX_CYCLE_NON_GBR" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="10">10</option>
			<option value="20">20</option>
			<option value="32">32</option>
			<option value="40">40</option>
			<option value="64">64</option>
			<option value="80">80</option>
			<option value="128">128</option>
			<option value="160">160</option>
			<option value="256">256</option>
			<option value="320">320</option>
			<option value="512">512</option>
			<option value="640">640</option>
			<option value="1024">1024</option>
			<option value="1280">1280</option>
			<option value="2048">2048</option>
			<option value="2560">2560</option>
		</select>
	</li>
	<%-- DRX重传定时器 --%>
	<li>
		<label for="LTE_DRX_RETRANSMISSION_TIMER_name">${LTE_DRX_RETRANSMISSION_TIMER_name }</label>
		<select id="LTE_DRX_RETRANSMISSION_TIMER_name" name="LTE_DRX_RETRANSMISSION_TIMER" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="1">1</option>
			<option value="2">2</option>
			<option value="4">4</option>
			<option value="6">6</option>
			<option value="8">8</option>
			<option value="16">16</option>
			<option value="24">24</option>
			<option value="33">33</option>
		</select>
	</li>
</ul>
