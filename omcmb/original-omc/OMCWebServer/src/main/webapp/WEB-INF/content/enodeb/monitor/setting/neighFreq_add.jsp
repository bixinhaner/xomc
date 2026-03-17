<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp"%>
<style>
	#freqEditPanel .el-form-item {
		width: 85% !important;
	}
</style>
<div id='freqEditPanel'>
	<el-form ref="freqForm" :model="freqForm" :rules="freqRules" 
		label-position="top" label-width="120" style='width:100%;height:100%;' inline hide-required-asterisk="true">
		
		<div style='height:37px;border-bottom:1px solid #EEE;line-height:37px;font-weight:bold;padding-left:10px;font-size:14px;'>
				Add Neigh Freq List
				<div class="circleIcon placeholder-bt" style="top: 4px;right:40px;" placeholder="Goback">		
					<span class="el-icon el-icon-circle-goback" @click='closeAddFreq'></span>
				</div>
		</div>
		<el-collapse v-model="activeNames">
			<!-- Quick Setting -->
			<el-collapse-item name="freq">
				<template slot='title'>
					<p style="display:inline-block;margin-left:40px;">
						<span class="title-icon" style="vertical-align:sub"></span>
						<span style="font-size:14px;font-weight:bold">Neigh Freq Setting</span>
					</p>
				</template>
				<div style="">
					<div class='list-cls'>
						<div class='list-item-cls'>
							<el-form-item label="Enable" prop="freqEnable">
								<el-switch v-model="freqForm.freqEnable" active-value="true" inactive-value="false" active-color="#4D84FF" inactive-color="#BDC1C6"></el-switch>
							</el-form-item>
							<el-form-item label="EARFCN" prop="EARFCN" class='validate-item'>
								<el-input v-model="freqForm.EARFCN">
									<template slot="append">Range:0~65535</template>
								</el-input>
							</el-form-item>
							<el-form-item label="Q-OffsetRange" prop="QOffsetRange">
								<el-select v-model="freqForm.QOffsetRange">
									<el-option v-for="item in QOffsetList" :label="item.text" :value="item.value" :key="item.value"></el-option>
								</el-select>
							</el-form-item>
							<el-form-item label="PMax" prop="PMax" class="validate-item">
								<el-input v-model="freqForm.PMax">
									<template slot="append">Range:-30~33</template>
								</el-input>
							</el-form-item>
							<el-form-item label="ReselThreshHigh" prop="ReselThreshHigh" class='validate-item'>
								<el-input v-model="freqForm.ReselThreshHigh">
									<template slot="append">Range:0~31</template>
								</el-input>
							</el-form-item>
						</div>
						<div class='list-item-cls' style='padding-top:45px;'>
							<el-form-item label="qRxLevMinSib5" prop="qRxLevMinSib5" class="validate-item">
								<el-input v-model="freqForm.qRxLevMinSib5">
									<template slot="append">Range:-70~-22</template>
								</el-input>
							</el-form-item>
							<el-form-item label="tReselectionEutra" prop="tReselectionEutra" class="validate-item">
								<el-input v-model="freqForm.tReselectionEutra">
									<template slot="append">Range:0~7</template>
								</el-input>
							</el-form-item>
							<el-form-item label="ReselThreshLow" prop="ReselThreshLow" class="validate-item">
								<el-input v-model="freqForm.ReselThreshLow">
									<template slot="append">Range:0~31</template>
								</el-input>
							</el-form-item>
							<el-form-item label="ReselectionPriority" prop="ReselectionPriorityFreq" class="validate-item">
								<el-input v-model="freqForm.ReselectionPriorityFreq">
									<template slot="append">Range:0~7</template>
								</el-input>
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
		el:'#freqEditPanel',
		data(){
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
			return{
				freqForm:{
					freqEnable: 'true',
					EARFCN:'',
					QOffsetRange:'',
					PMax:'',
					ReselThreshHigh:'',
					qRxLevMinSib5:'',
					tReselectionEutra:'',
					ReselThreshLow:'',
					ReselectionPriorityFreq:'',
					Index:''
				},
				freqRules:{
					EARFCN:[{validator:validateRange,min:0,max:65535}],
					QOffsetRange:[{required:true,message:'required'}],
					PMax:[{validator:validateRange,min:-30,max:33}],
					ReselThreshHigh:[{validator:validateRange,min:0,max:31}],
					qRxLevMinSib5:[{validator:validateRange,min:-70,max:-22}],
					tReselectionEutra:[{validator:validateRange,min:0,max:7}],
					ReselThreshLow:[{validator:validateRange,min:0,max:31}],
					ReselectionPriorityFreq:[{validator:validateRange,min:0,max:7}]
				},
				activeNames:['freq'],
				QOffsetList:[{"text":"dB-24","value":"-24"},{"text":"dB-22","value":"-22"},{"text":"dB-20","value":"-20"},{"text":"dB-18","value":"-18"},{"text":"dB-16","value":"-16"},{"text":"dB-14","value":"-14"},{"text":"dB-12","value":"-12"},{"text":"dB-10","value":"-10"},{"text":"dB-8","value":"-8"},{"text":"dB-6","value":"-6"},{"text":"dB-5","value":"-5"},{"text":"dB-4","value":"-4"},{"text":"dB-3","value":"-3"},{"text":"dB-2","value":"-2"},{"text":"dB-1","value":"-1"},{"text":"dB0","value":"0"},{"text":"dB1","value":"1"},{"text":"dB2","value":"2"},{"text":"dB3","value":"3"},{"text":"dB4","value":"4"},{"text":"dB5","value":"5"},{"text":"dB6","value":"6"},{"text":"dB8","value":"8"},{"text":"dB10","value":"10"},{"text":"dB12","value":"12"},{"text":"dB14","value":"14"},{"text":"dB16","value":"16"},{"text":"dB18","value":"18"},{"text":"dB20","value":"20"},{"text":"dB22","value":"22"},{"text":"dB24","value":"24"}],
				
			}
		},
		methods:{
			closeAddFreq(){
				lteVm.showNeigh = false;
			},
			init(){
				if(lteVm.operType == 'edit'){
					Object.assign(this.freqForm,lteVm.rowDataFreq)
				}
			},
			save(){
				var vm = this;
				vm.$refs.freqForm.validate(function(valid){
					if(valid){
						var row = Object.assign({operateType: lteVm.operType}, vm.freqForm),
							list = lteVm.lteForm.NeighFreqList.map(function(item){
								return item.Index+'';
							}),
							index = '1';

						if(lteVm.rowDataCell.operateType == 'add') {
							row.operateType = 'add';
						}

						if(lteVm.operType == 'add'){
							for(var i=1;i<=16;i++) {
								if(list.includes(i+'')) {

								}else {
									index = i + '';
									break;
								}
							}
							row.Index = index;
							lteVm.lteForm.NeighFreqList.push(row);
						}else{
							Object.assign(lteVm.rowDataFreq, row);
						}
						lteVm.showNeigh = false;
					}
				})
			}
		},
		mounted(){
			this.init();
			eventBus.$off('save-freq').$on('save-freq',this.save);
		}
		
	})
</script>