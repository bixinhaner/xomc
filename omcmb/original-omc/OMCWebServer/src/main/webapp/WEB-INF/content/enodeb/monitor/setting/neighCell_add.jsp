<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp"%>
<style>
	#cellEditPanel .el-form-item {
		width: 85% !important;
	}
</style>
<div id='cellEditPanel'>
	<el-form ref="cellForm" :model="cellForm" :rules="cellRules" 
		label-position="top" label-width="120" style='width:100%;height:100%;' inline hide-required-asterisk="true">
		
		<div style='height:37px;border-bottom:1px solid #EEE;line-height:37px;font-weight:bold;padding-left:10px;font-size:14px;'>
				Add Neigh Cell List
				<div class="circleIcon placeholder-bt" style="top: 4px;right:40px;" placeholder="<%=rb.getString("FanHui")%>">		
					<span class="el-icon el-icon-circle-goback" @click='closeAddFreq'></span>
				</div>
		</div>
		<el-collapse v-model="activeNames">
			<!-- Quick Setting -->
			<el-collapse-item name="freq">
				<template slot='title'>
					<p style="display:inline-block;margin-left:40px;">
						<span class="title-icon" style="vertical-align:sub"></span>
						<span style="font-size:14px;font-weight:bold">Neigh Cell Setting</span>
					</p>
				</template>
				<div style="">
					<div class='list-cls'>
						<div class='list-item-cls'>
							<el-form-item label="Enable" prop="cellEnable">
								<el-switch v-model="cellForm.cellEnable" active-value="true" inactive-value="false" active-color="#4D84FF" inactive-color="#BDC1C6"></el-switch>
							</el-form-item>

							<el-form-item :label="isBaiBLQ?'ECI':'Cell ID'" prop="CellID" class="validate-item">
								<el-input v-model="cellForm.CellID" :disabled="isActive && isEdit">
									<template slot="append">Range:0~268435455</template>
								</el-input>
							</el-form-item>

							<el-form-item label="EARFCN" prop="CellEARFCN" class="validate-item">
								<el-input v-model="cellForm.CellEARFCN" :disabled="isActive && isEdit">
									<template slot="append">Range:0~65535</template>
								</el-input>
							</el-form-item>
							<el-form-item label="QOffset" prop="QOffset">
								<el-select v-model="cellForm.QOffset">
									<el-option v-for="item in offsetList" :label="item.text" :value="item.value" :key="item.value"></el-option>
								</el-select>
							</el-form-item>
							<el-form-item v-show="isBaiBLQ && !isMLQ" label="X2 Flag (<%=rb.getString("SheBeiBuZhiChi")%>)" prop="x2Flag">
								<el-select v-model="cellForm.x2Flag" :disabled="isEdit">
									<el-option label=" " value=""></el-option>
									<el-option label="SON" value="0"></el-option>
									<el-option label="Manual" value="1"></el-option>
								</el-select>
							</el-form-item>
							<el-form-item v-show="isBaiBLQ && !isMLQ && cellForm.x2Flag == '1'" label="X2 IP (<%=rb.getString("SheBeiBuZhiChi")%>)" prop="x2IP">
								<el-input v-model="cellForm.x2IP"></el-input>
							</el-form-item>
						</div>
						<div class='list-item-cls' style='padding-top:45px;'>
							<el-form-item label="PLMN" prop="PLMN" class="validate-item">
								<el-input v-model="cellForm.PLMN" :disabled="isActive && isEdit">
									<template slot="append">Range:5~6 digits</template>
								</el-input>
							</el-form-item>
							<el-form-item label="PCI" prop="PCI" class="validate-item">
								<el-input v-model="cellForm.PCI">
									<template slot="append">Range:0~503</template>
								</el-input>
							</el-form-item>
							<el-form-item label="TAC" prop="TAC" class="validate-item">
								<el-input v-model="cellForm.TAC">
									<template slot="append">Range:0~65535</template>
								</el-input>
							</el-form-item>
							<el-form-item label="CIO" prop="CIO">
								<el-select v-model="cellForm.CIO">
									<el-option v-for="item in cioList" :label="item.text" :value="item.value" :key="item.value"></el-option>
								</el-select>
							</el-form-item>
							<el-form-item v-show="isBaiBLQ && !isMLQ" label="eNodeB Type" prop="enbType">
								<el-select v-model="cellForm.enbType">
									<el-option label="Home" value="1"></el-option>
									<el-option label="Macro" value="0"></el-option>
								</el-select>
							</el-form-item>
						</div>
					</div>
				</div>
			</el-collapse-item>
		</el-collapse>
	</el-form>
</div>
<script>
	new Vue({
		el:'#cellEditPanel',
		data(){
			var vm = this;
			var validateRange = (rule,value,callback)=>{
				var min = rule.min;
				var max = rule.max;
				var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;
				if(value == '' || !reg.test(value) || value < min || value > max){
					callback(new Error('format error'))
				}else{
					callback();
				}
			}
			var validatePlmn = (rule,value,callback) => {
				var minVal = rule.min;
				var maxVal = rule.max;
				var reg = /^-?\d+$/;
				if(value == ""){
					callback(new Error('format error'));
				}else if(reg.test(value) && (value >= parseInt(minVal)) && (value <= parseInt(maxVal)) && value.length<=6 && value.length >=5){
					callback();
				}else{
					callback(new Error('format error'));
				}
			}
			var validateX2IP = (rule, value, cb)=>{
					var trimVal = value.trim(),
						x2flag = vm.cellForm.x2Flag;

					if(x2flag == '1') { // 手动模式
						if(trimVal !== '') {
							if(isIPv4(trimVal)) {
								cb();
							}else{
								cb('Invalid IP address')
							}
						}else {
							cb('Required');
						}
					}else {
						cb();
					}
				};

			return{
				isBaiBLQ: lteVm.isBaiblq,
				isMLQ: lteVm.isMLQ,

				cellForm:{
					cellEnable: 'true',
					PLMN:'',
					CellID:'',
					CellEARFCN:'',
					PCI:'',
					QOffset:'',
					CIO:'',
					TAC:'',
					CellIndex:'',

					enbType: '',
					x2Flag: '',
					x2IP: '',
					neighborType: '2',
					x2Status: '',
				},
				cellRules:{
					CellID:[{validator:validateRange,min:0,max:268435455}],
					PLMN:[{validator:validatePlmn,min:0,max:999999}],
					CellEARFCN:[{validator:validateRange,min:0,max:65535}],
					PCI:[{validator:validateRange,min:0,max:503}],
					QOffset:[{required:true,message:'Required'}],
					CIO:[{required:true,message:'Required'}],
					TAC:[{validator:validateRange,min:0,max:65535}],
					x2IP:[{validator:validateX2IP}]
				},
				activeNames:['freq'],
				casts:{
					
				},
				offsetList:[{"text":"-24","value":"-24"},{"text":"-22","value":"-22"},{"text":"-20","value":"-20"},{"text":"-18","value":"-18"},{"text":"-16","value":"-16"},{"text":"-14","value":"-14"},{"text":"-12","value":"-12"},{"text":"-10","value":"-10"},{"text":"-8","value":"-8"},{"text":"-6","value":"-6"},{"text":"-5","value":"-5"},{"text":"-4","value":"-4"},{"text":"-3","value":"-3"},{"text":"-2","value":"-2"},{"text":"-1","value":"-1"},{"text":"0","value":"0"},{"text":"1","value":"1"},{"text":"2","value":"2"},{"text":"3","value":"3"},{"text":"4","value":"4"},{"text":"5","value":"5"},{"text":"6","value":"6"},{"text":"8","value":"8"},{"text":"10","value":"10"},{"text":"12","value":"12"},{"text":"14","value":"14"},{"text":"16","value":"16"},{"text":"18","value":"18"},{"text":"20","value":"20"},{"text":"22","value":"22"},{"text":"24","value":"24"}],
				cioList:[{"text":"-24","value":"-24"},{"text":"-22","value":"-22"},{"text":"-20","value":"-20"},{"text":"-18","value":"-18"},{"text":"-16","value":"-16"},{"text":"-14","value":"-14"},{"text":"-12","value":"-12"},{"text":"-10","value":"-10"},{"text":"-8","value":"-8"},{"text":"-6","value":"-6"},{"text":"-5","value":"-5"},{"text":"-4","value":"-4"},{"text":"-3","value":"-3"},{"text":"-2","value":"-2"},{"text":"-1","value":"-1"},{"text":"0","value":"0"},{"text":"1","value":"1"},{"text":"2","value":"2"},{"text":"3","value":"3"},{"text":"4","value":"4"},{"text":"5","value":"5"},{"text":"6","value":"6"},{"text":"8","value":"8"},{"text":"10","value":"10"},{"text":"12","value":"12"},{"text":"14","value":"14"},{"text":"16","value":"16"},{"text":"18","value":"18"},{"text":"20","value":"20"},{"text":"22","value":"22"},{"text":"24","value":"24"}],
				
				
			}
		},
		computed: {
			isActive() {
				return this.cellForm.cellEnable == 'true'
			},
			isEdit() {
				return lteVm.operType == 'edit';
			}
		},
		methods:{
			closeAddFreq(){
				lteVm.showNeigh = false;
			},
			init(){
				if(lteVm.operType == 'edit'){
					Object.assign(this.cellForm,lteVm.rowDataCell)
				}
			},
			save(){
				var vm = this;
				vm.$refs.cellForm.validate(function(valid){
					if(valid){
						var codes = ['cellEnable','PLMN','CellID','CellEARFCN','PCI','QOffset','CIO','TAC','CellIndex'];

						if(vm.isBaiBLQ && !vm.isMLQ) {// BaiBLQ设备类型字段赋值
							codes = ['cellEnable','PLMN','CellID','CellEARFCN','PCI','QOffset','CIO','TAC','CellIndex',
									'enbType', 'x2Flag', 'x2IP', 'neighborType'];
						}

						var row = {operateType: lteVm.operType};

						if(lteVm.rowDataCell.operateType == 'add') {
							row.operateType = 'add';
						}

						codes.map(function(code){
							row[code] = vm.cellForm[code];
						})

						var list = lteVm.lteForm.NeighCellList.map(function(item){
								return item.CellIndex+'';
							}),
							index = '1';

						if(lteVm.operType == 'add'){
							for(var i=1;i<=160;i++) {
								if(list.includes(i+'')) {

								}else {
									index = i + '';
									break;
								}
							}
							
							// 自动模式，清除x2IP
							if(row.x2Flag !== '1') {
								try{ delete row['x2IP']; }catch(e){}
							}
							
							row.CellIndex = index;
							lteVm.lteForm.NeighCellList.push(row);
						}else{
							['x2Flag', 'x2IP'].map(function(code){
								if(lteVm.rowDataCell[code] === row[code]) {
									try{ delete row[code]; }catch(e){}
								}
							});
							
							// 自动模式，清除x2IP
							if(row.x2Flag !== '1') {
								try{ delete row['x2IP']; }catch(e){}
							}

							Object.assign(lteVm.rowDataCell, row)
						}

						lteVm.showNeigh = false;
					}
				})
			}
		},
		mounted(){
			this.init();
			eventBus.$off('save-cell').$on('save-cell',this.save);
		}
	})
</script>