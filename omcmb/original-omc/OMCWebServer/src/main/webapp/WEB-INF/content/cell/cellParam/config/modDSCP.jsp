<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<%@ taglib uri="http://java.sun.com/jsp/jstl/core" prefix="c" %>
<ul id="paramNodesUl" class="paramNodesUl">
	<c:if test='${hardwareVersion == "EA4.0DUAL"}'>
		<li>
			<label for="LTE_HALOB_DSCP_SWITCH_name">${LTE_HALOB_DSCP_SWITCH_name }</label>
			<select id="LTE_HALOB_DSCP_SWITCH_name" name="LTE_HALOB_DSCP_SWITCH" class="border border-box" onblur="createMML();">
				<option value=""></option>
				<option value="1">ON</option>
				<option value="0">OFF</option>
			</select>
		</li>
	</c:if>

	<li>
		<label for="LTE_HALOB_DSCP1_name">${LTE_HALOB_DSCP1_name }</label>
		<select id="LTE_HALOB_DSCP1_name" name="LTE_HALOB_DSCP1" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="0">0</option>
			<option value="8">8</option>
			<option value="10">10</option>
			<option value="12">12</option>
			<option value="14">14</option>
			<option value="16">16</option>
			<option value="18">18</option>
			<option value="20">20</option>
			<option value="22">22</option>
			<option value="24">24</option>
			<option value="26">26</option>
			<option value="28">28</option>
			<option value="30">30</option>
			<option value="32">32</option>
			<option value="34">34</option>
			<option value="36">36</option>
			<option value="38">38</option>
			<option value="40">40</option>
			<option value="44">44</option>
			<option value="46">46</option>
			<option value="48">48</option>
			<option value="56">56</option>
		</select>
	</li>

	<li>
		<label for="LTE_HALOB_DSCP2_name">${LTE_HALOB_DSCP2_name }</label>
		<select id="LTE_HALOB_DSCP2_name" name="LTE_HALOB_DSCP2" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="0">0</option>
			<option value="8">8</option>
			<option value="10">10</option>
			<option value="12">12</option>
			<option value="14">14</option>
			<option value="16">16</option>
			<option value="18">18</option>
			<option value="20">20</option>
			<option value="22">22</option>
			<option value="24">24</option>
			<option value="26">26</option>
			<option value="28">28</option>
			<option value="30">30</option>
			<option value="32">32</option>
			<option value="34">34</option>
			<option value="36">36</option>
			<option value="38">38</option>
			<option value="40">40</option>
			<option value="44">44</option>
			<option value="46">46</option>
			<option value="48">48</option>
			<option value="56">56</option>
		</select>
	</li>

	<li>
		<label for="LTE_HALOB_DSCP3_name">${LTE_HALOB_DSCP3_name }</label>
		<select id="LTE_HALOB_DSCP3_name" name="LTE_HALOB_DSCP3" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="0">0</option>
			<option value="8">8</option>
			<option value="10">10</option>
			<option value="12">12</option>
			<option value="14">14</option>
			<option value="16">16</option>
			<option value="18">18</option>
			<option value="20">20</option>
			<option value="22">22</option>
			<option value="24">24</option>
			<option value="26">26</option>
			<option value="28">28</option>
			<option value="30">30</option>
			<option value="32">32</option>
			<option value="34">34</option>
			<option value="36">36</option>
			<option value="38">38</option>
			<option value="40">40</option>
			<option value="44">44</option>
			<option value="46">46</option>
			<option value="48">48</option>
			<option value="56">56</option>
		</select>
	</li>
	<li>
		<label for="LTE_HALOB_DSCP4_name">${LTE_HALOB_DSCP4_name }</label>
		<select id="LTE_HALOB_DSCP4_name" name="LTE_HALOB_DSCP4" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="0">0</option>
			<option value="8">8</option>
			<option value="10">10</option>
			<option value="12">12</option>
			<option value="14">14</option>
			<option value="16">16</option>
			<option value="18">18</option>
			<option value="20">20</option>
			<option value="22">22</option>
			<option value="24">24</option>
			<option value="26">26</option>
			<option value="28">28</option>
			<option value="30">30</option>
			<option value="32">32</option>
			<option value="34">34</option>
			<option value="36">36</option>
			<option value="38">38</option>
			<option value="40">40</option>
			<option value="44">44</option>
			<option value="46">46</option>
			<option value="48">48</option>
			<option value="56">56</option>
		</select>
	</li>
	<li>
		<label for="LTE_HALOB_DSCP5_name">${LTE_HALOB_DSCP5_name }</label>
		<select id="LTE_HALOB_DSCP5_name" name="LTE_HALOB_DSCP5" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="0">0</option>
			<option value="8">8</option>
			<option value="10">10</option>
			<option value="12">12</option>
			<option value="14">14</option>
			<option value="16">16</option>
			<option value="18">18</option>
			<option value="20">20</option>
			<option value="22">22</option>
			<option value="24">24</option>
			<option value="26">26</option>
			<option value="28">28</option>
			<option value="30">30</option>
			<option value="32">32</option>
			<option value="34">34</option>
			<option value="36">36</option>
			<option value="38">38</option>
			<option value="40">40</option>
			<option value="44">44</option>
			<option value="46">46</option>
			<option value="48">48</option>
			<option value="56">56</option>
		</select>
	</li>
	<li>
		<label for="LTE_HALOB_DSCP6_name">${LTE_HALOB_DSCP6_name }</label>
		<select id="LTE_HALOB_DSCP6_name" name="LTE_HALOB_DSCP6" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="0">0</option>
			<option value="8">8</option>
			<option value="10">10</option>
			<option value="12">12</option>
			<option value="14">14</option>
			<option value="16">16</option>
			<option value="18">18</option>
			<option value="20">20</option>
			<option value="22">22</option>
			<option value="24">24</option>
			<option value="26">26</option>
			<option value="28">28</option>
			<option value="30">30</option>
			<option value="32">32</option>
			<option value="34">34</option>
			<option value="36">36</option>
			<option value="38">38</option>
			<option value="40">40</option>
			<option value="44">44</option>
			<option value="46">46</option>
			<option value="48">48</option>
			<option value="56">56</option>
		</select>
	</li>
	<li>
		<label for="LTE_HALOB_DSCP7_name">${LTE_HALOB_DSCP7_name }</label>
		<select id="LTE_HALOB_DSCP7_name" name="LTE_HALOB_DSCP7" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="0">0</option>
			<option value="8">8</option>
			<option value="10">10</option>
			<option value="12">12</option>
			<option value="14">14</option>
			<option value="16">16</option>
			<option value="18">18</option>
			<option value="20">20</option>
			<option value="22">22</option>
			<option value="24">24</option>
			<option value="26">26</option>
			<option value="28">28</option>
			<option value="30">30</option>
			<option value="32">32</option>
			<option value="34">34</option>
			<option value="36">36</option>
			<option value="38">38</option>
			<option value="40">40</option>
			<option value="44">44</option>
			<option value="46">46</option>
			<option value="48">48</option>
			<option value="56">56</option>
		</select>
	</li>
	<li>
		<label for="LTE_HALOB_DSCP8_name">${LTE_HALOB_DSCP8_name }</label>
		<select id="LTE_HALOB_DSCP8_name" name="LTE_HALOB_DSCP8" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="0">0</option>
			<option value="8">8</option>
			<option value="10">10</option>
			<option value="12">12</option>
			<option value="14">14</option>
			<option value="16">16</option>
			<option value="18">18</option>
			<option value="20">20</option>
			<option value="22">22</option>
			<option value="24">24</option>
			<option value="26">26</option>
			<option value="28">28</option>
			<option value="30">30</option>
			<option value="32">32</option>
			<option value="34">34</option>
			<option value="36">36</option>
			<option value="38">38</option>
			<option value="40">40</option>
			<option value="44">44</option>
			<option value="46">46</option>
			<option value="48">48</option>
			<option value="56">56</option>
		</select>
	</li>
	<li>
		<label for="LTE_HALOB_DSCP9_name">${LTE_HALOB_DSCP9_name }</label>
		<select id="LTE_HALOB_DSCP9_name" name="LTE_HALOB_DSCP9" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="0">0</option>
			<option value="8">8</option>
			<option value="10">10</option>
			<option value="12">12</option>
			<option value="14">14</option>
			<option value="16">16</option>
			<option value="18">18</option>
			<option value="20">20</option>
			<option value="22">22</option>
			<option value="24">24</option>
			<option value="26">26</option>
			<option value="28">28</option>
			<option value="30">30</option>
			<option value="32">32</option>
			<option value="34">34</option>
			<option value="36">36</option>
			<option value="38">38</option>
			<option value="40">40</option>
			<option value="44">44</option>
			<option value="46">46</option>
			<option value="48">48</option>
			<option value="56">56</option>
		</select>
	</li>
</ul>

<script type="text/javascript"> 
$(function(){
	$('#LTE_HALOB_ENABLE_STATE_name').bind('change',function(){
		if (this.value == "1") {
			$("#halobModeLi").css("display", "inline-block");
		} else {
			$("#halobModeLi").css("display", "none");
		}
	});
});
</script>