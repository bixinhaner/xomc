
<%@ page import="java.util.Locale"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page contentType="text/html;charset=UTF-8"%>

<style>
    .sub-title {
        margin-top: 15px;
    }
    .sub-title .title-text {
        font-size: 12px;
    }
    .sub-title .title-icon {
        position: relative;
        background: none;
    }
    .sub-title .title-icon::before {
        position: absolute;
        content: '.';
        left: 10px;
        top: -26px;
        font-size: 40px;
    }

    .splite-line {
        opacity: 0.2;
        margin-bottom: 30px;
    }
    .flex-wrap {
        flex-wrap: wrap;
    }
    .flex-wrap .el-form-item {
        width: 90%;
    }
    .flex-wrap .half {
        width: 46%;
    }
    .flex-wrap .third {
        width: 29%;
    }
    .el-select .el-input__inner {
        height: 26px !important;
    }
    .label-padding .el-form-item__label {
    	padding-left: 10px;
    }
    .three-item {
    	flex-wrap: wrap;
    }
    .three-item .el-form-item {
    	margin-bottom: 30px;
    	width: 30%;
    }
    .label-padding:last-child {
    	margin-bottom: 0px !important;
    	border: none !important;
    }
</style>

<div id="apn_setting_ctn" style="padding: 20px 40px;">
    <el-form ref="form" :model="form" :rules="rules" label-position="left" label-width="150">
        <div class="group-title not-extend margin-left-px sub-title">
            <span class="title-icon"></span>
            <span class="title-text">LGW Settings</span>
        </div>
        <div class="flex-form flex-wrap" style="padding: 15px 0px 0px 25px;">
        	<el-form-item label="Automatic Configuration" label-width="200" prop="automaticEnable">
        		<el-switch v-model="form.automaticEnable" active-value="true" inactive-value="false"></el-switch>
        	</el-form-item>
        	<el-form-item label="LGW Mode" label-width="200" prop="lgwMode">
        		<el-select v-model="form.lgwMode" :disabled="form.automaticEnable == 'true'">
                    <el-option label="NAT" value="0" :disabled="isBridge"></el-option>
                    <el-option label="Bridge" value="2"></el-option>
                </el-select>
        	</el-form-item>
        	<el-form-item label="eNB MTU" label-width="200" prop="mtu">
        		<el-input v-model="form.mtu" placeholder="Range: 700 - 1600"></el-input>
        	</el-form-item>
        	<el-form-item label="CPE MTU" label-width="200" prop="cpeMtu">
        		<el-input v-model="form.cpeMtu" placeholder="Range: 700 - 1600"></el-input>
        	</el-form-item>
        </div>
        <div class="group-title not-extend margin-left-px sub-title">
            <span class="title-icon"></span>
            <span class="title-text">Global APN Settings</span>
        </div>
        <div class="flex-form flex-wrap" style="padding: 15px 0px 0px 25px;">
            <div class="el-form-item" style="width: 90%;min-width: 1200px;">
                <div style="font-size: 14px;">APN List</div>
                <div class="group-bg-color" style="display: flex;background-color: #fff;padding-top: 0px;">
                    <div style="padding-top: 20px;width: 100%;">
                        <div v-for="(row, idx) in form.apnConfig" class="flex-form label-padding" style="margin-bottom: 30px;width: 100%;border-bottom: 1px solid #E8E8E8;flex-wrap: nowrap;">
                            <span style="padding: 8px 20px 0 15px;font-weight: bold;">APN{{row.apnOrder}}</span>
                            <div class="flex-form three-item" style="width: 100%;">
	                            <el-form-item label="APN Name" label-width="140" class="readonly-cls">
	                                <el-input v-model="row.apnName" style="width: 200px;" readonly></el-input>
	                            </el-form-item>
	
	                            <el-form-item label="APN Type" label-width="140">
	                                <el-select v-model="row.apnTypeEnb" class="auto-width" style="width: 200px;" 
	                                	@change="function(value){
                                            if(value == 'tunnel') {
                                            	row.apnType = '2';
                                            	if(row.vlanId) {
	                                                //var list = row.vlanId.split(',');
	
	                                                //row.untaggedVlan = list[0];
	                                            }

                                                row.untaggedVlan = '';
                                                row.vlanId = row.taggedVlan;
                                            }else if(row.apnType == '2'){
                                            	row.apnType = '1';
                                            }
                                            
                                            linkTunnel();
                                        }">
	                                    <el-option label="Normal" value="normal"></el-option>
	                                    <el-option label="Tunnel" value="tunnel"></el-option>
	                                </el-select>
	                            </el-form-item>
	                            
	                            <el-form-item v-show="row.apnType != '2'" :label="['0','1','3'].includes(row.apnType)?'VLAN ID':'VLAN List'" label-width="140"
	                                :prop="'apnConfig.'+idx+'.vlanId'"
	                                :key="row.apnOrder+'_vlanId'"
	                                :rules="{
	                                    validator: function(rule, val, cb){
	                                        var reg = /^\d{1,}$/,
	                                            ids = (val+'').split(','),
	                                            bool = true;
                                            // 校验输入值是否满足 10-4094范围
	                                        ids.map(function(item){
	                                            var trimVal = item.trim();
	                                            
	                                            if(!reg.test(trimVal)) {
	                                                bool = false;
	                                            }
	                                            
	                                            if((trimVal<1 && ![0,'0'].includes(trimVal)) || trimVal>4094) {
	                                                bool = false;
	                                            }
	                                        });

	                                        if(row.apnType == '2') {// tunnel mode, vlan List hide
                                                cb();
                                            }else {
                                                if(val || [0,'0'].includes(val)) {
                                                    if(['2'].includes(row.apnType) && (!bool || ids.length>3)) {
                                                        if(val && ![0,'0'].includes(val)) cb('Allows up to 3 comma-separated numbers,range of each number is from 0 to 4094');
                                                        else cb();
                                                    }else if(['0','1','3'].includes(row.apnType) && (!bool || ids.length>1)) {
                                                        if(val && ![0,'0'].includes(val)) {
                                                            cb('Single number is from 0 to 4094');
                                                        }else {
                                                            var isL3Empty = false,
                                                                l3LanId = form.apnConfig.filter(function(m){
                                                                    return ['0','1','3'].includes(m.apnType)
                                                                }),
                                                                l3EmptyLanId = form.apnConfig.filter(function(m){
                                                                    return ['0','1','3'].includes(m.apnType) && m.vlanId.trim() === '';
                                                                });
        
                                                            if(['0','1','3'].includes(row.apnType) && l3EmptyLanId.length == l3LanId.length) {
                                                                cb('VLAN ID of L3 should`t all empty');
                                                            }else {
                                                                cb();
                                                            }
                                                        }
                                                    }else {
                                                        var enableVlans = form.apnConfig.filter(function(m){
                                                                return m.apnOrder != row.apnOrder && m.apnType == '2'; // 是否和tunnel的存在id重复
                                                            }),
                                                            otherVlans = [],
                                                            selfVlans = [],
                                                            isRepeated = false;
        
                                                        enableVlans.map(function(o){
                                                            if(o.vlanId.length) {
                                                                (o.vlanId+'').split(',').map(function(m){
                                                                    otherVlans.push(m);
                                                                });
                                                            }
                                                        });
        
                                                        ids.map(function(m){
                                                            if(!selfVlans.includes(m)) {
                                                                selfVlans.push(m);
                                                            }
                                                        });
        
                                                        ids.map(function(m){
                                                            if(otherVlans.includes(m)) isRepeated = true;
                                                        });
        
                                                        if(ids.length > selfVlans.length) isRepeated = true;
        
                                                        if(isRepeated) {
                                                            cb('The Vlan ID in total APNs of ENB cannot be overlap');
                                                        }else {
                                                            cb();
                                                        }
                                                    } 
                                                }else {
                                                    cb();
                                                }
                                            }
	                                    }
	                                }">
	                                <el-input v-model="row.vlanId" style="width: 200px;" @change="function(value){
                                            if(value.length && row.apnType == '2') {
                                                var list = value.split(',');
                                                
                                                //row.untaggedVlan = list[0];
                                            }
                                        }">
                                    </el-input>
	                            </el-form-item>
	                        	
	                        	<div style="width: 98%; display: flex;flex-wrap: wrap;border: 1px solid #E8E8E8;padding-top: 19px;background-color: #f9f9f9;margin-bottom: 10px;">
		                        	<el-form-item label="CPE Default Setting" label-width="160"></el-form-item>
		
		                            <el-form-item label="Bearer Type" label-width="140" style="margin-left: 6px;">
		                                <el-select v-model="row.apnType" class="auto-width" style="width: 200px;" 
		                                	@change="function(value){
	                                            linkTunnel(value);

		                                        if(row.apnType != '2' && row.apnType != '3') {
		                                       	   row.untaggedVlan = '';
		                                       	   row.taggedVlan = '';
		                                        }
                                                
                                                $refs.form.validateField('apnConfig.'+idx+'.taggedVlan');
	                                        }">
		                                    <el-option label="L3-MGMT" value="0" :disabled="row.apnTypeEnb == 'tunnel'"></el-option>
		                                    <el-option label="L3-NAT" value="1" :disabled="row.apnTypeEnb == 'tunnel'"></el-option>
		                                    <el-option label="L2-TUNNEL" value="2" :disabled="row.apnTypeEnb != 'tunnel'"></el-option>
		                                    <el-option label="L2-BRIDGE" value="3" :disabled="row.apnTypeEnb == 'tunnel'"></el-option>
		                                </el-select>
		                            </el-form-item>
		                        	
		                        	<el-form-item v-show="row.apnType == '2'" label="Untagged VLAN" label-width="140" style="margin-left: 3px;"
		                        		:prop="'apnConfig.'+idx+'.untaggedVlan'"
		                                :key="row.apnOrder+'_untaggedVlan'"
		                                :rules="{
		                                    validator: function(rule, val, cb){
		                                    	var enbVlans = (row.vlanId||'').split(','),
	                                                enableVlans = form.apnConfig.filter(function(m){
	                                                    return m.apnOrder != row.apnOrder && ['2'].includes(m.apnType);
	                                                }),
                                                    otherVlans = [],
	                                                isRepeated = false;

                                                enableVlans.map(function(o){
	                                                if(o.untaggedVlan || ['0'].includes(o.untaggedVlan)){
	                                                    otherVlans.push(o.untaggedVlan);
	                                                }
	                                            })

                                                if(otherVlans.includes(val)) isRepeated = true;
	                                            
	                                            if(row.apnType == '2') {
	                                                if(!isNaN(val) && (val-1>=0 && val-4094<=0) || ['','0'].includes(val)) {
                                                        if(isRepeated) {
                                                            cb('The untagged Vlans in total APNs of CPE cannot be overlap');
                                                        }else {
                                                            var list = (row.taggedVlan||'').split(',');
                                                            
                                                            if(val && list.includes(val) && !['0'].includes(val)) {
                                                                cb('Untagged VLAN and tagged VLANs cannot be overlap');
                                                            }else {
                                                                cb();
                                                            }
                                                        }
	                                                }else {
	                                                    cb('Single number is from 0 to 4094');
	                                                }
	                                            }else {
	                                                cb();
	                                            }
		                                    }
		                                 }">
		                        		<el-input v-model="row.untaggedVlan" @change="function(val){
                                            $refs.form.validateField('apnConfig.'+idx+'.taggedVlan');

                                            if(row.apnType == '2') {
                                                var list = [];

                                                if(![null,undefined,''].includes(row.untaggedVlan)) {
                                                    row.untaggedVlan.split(',').map(function(m){
                                                        if(!list.includes(m)) list.push(m);
                                                    });
                                                }
                                                if(![null,undefined,''].includes(row.taggedVlan)) {
                                                    row.taggedVlan.split(',').map(function(m){
                                                        if(!list.includes(m)) list.push(m);
                                                    });
                                                }
                                                row.vlanId = list.join(',');
                                            }
                                        }"></el-input>
		                        	</el-form-item>
		                        	
		                        	<el-form-item v-show="row.apnType == '2'"></el-form-item>
		                        	<el-form-item v-show="row.apnType == '2'"></el-form-item>
		                        	
		                        	<el-form-item v-show="row.apnType != '0' && row.apnType != '1'" :label="row.apnType == '2'?'Tagged VLANs':'Tagged VLAN'" label-width="140" style="margin-left: 9px;"
		                        		:prop="'apnConfig.'+idx+'.taggedVlan'"
		                                :key="row.apnOrder+'_taggedVlan'"
		                                :rules="{
		                                    validator: function(rule, val, cb){
		                                    	var reg = /^\d{1,}$/,
	                                                enableVlans = form.apnConfig.filter(function(m){
	                                                    return m.apnOrder != row.apnOrder && ['2','3'].includes(m.apnType);
	                                                }),
	                                                ids = (val||'').split(','),
	                                                otherVlans = [],
	                                                selfVlans = [],
	                                                isRepeated = false,
	                                                bool = true;
	                                                
	                                            if(row.apnType == '0' || row.apnType == '1') {
	                                            	cb();
	                                            	return;
	                                            }

                                                var hasNat = form.apnConfig.filter(function(m){
	                                                    return m.apnOrder != row.apnOrder && ['1'].includes(m.apnType);
	                                                }).length > 0;
	
	                                            ids.map(function(item){
		                                            var trimVal = item.trim();
		                                            
		                                            if(trimVal.length && !reg.test(trimVal)) {
			                                            bool = false;
			                                        }
			                                        
			                                        if(trimVal<1 || trimVal>4094) {
			                                            bool = false;
			                                        }
			                                        
			                                        if(['2','3'].includes(row.apnType) && ['','0'].includes(trimVal)) {
			                                        	bool = true;

                                                        if(row.untaggedVlan - 1 >= 0 && ['0'].includes(trimVal)) {
                                                            bool = false;
                                                        }
                                                        //if(row.untaggedVlan == '' && trimVal == '') {
                                                        //    bool = false;
                                                        //}
                                                        // 包含L3-NAT，则tunnel的untagged vlan和tagged vlan不能同时为空
                                                        //if(hasNat && row.apnType == '2' && row.untaggedVlan == '' && (trimVal == '0' || trimVal == '')) {
                                                        //    bool = false;
                                                        //}
			                                        }
		                                        });
                                                
	                                            enableVlans.map(function(o){
	                                                (o.taggedVlan||'').split(',').map(function(m){
	                                                    otherVlans.push(m||'0');
	                                                });
	                                            });
	
	                                            ids.map(function(m){
	                                                if(!selfVlans.includes(m||'0')) {
	                                                    selfVlans.push(m||'0');
	                                                }
	                                            });
	
	                                            ids.map(function(m){
	                                                if(otherVlans.includes(m||'0') && !['','0'].includes(val)) isRepeated = true;
	                                            });
	
	                                            if(ids.length > selfVlans.length && !['','0'].includes(val)) isRepeated = true;
	
												if(bool) {
		                                            if(isRepeated) {
                                                        if(row.apnType == '3') {// bridge
		                                                    cb('The Tagged Vlan in total APNs of CPE cannot be overlap');
                                                        }else {
		                                                    cb('The Tagged Vlans in total APNs of CPE cannot be overlap');
                                                        }
		                                            }else {
	                                                    var list = (row.taggedVlan||'').split(',');
	
	                                                    if(val && list.includes(row.untaggedVlan) && row.untaggedVlan !== '0') {
	                                                        cb('Untagged VLAN and tagged VLANs cannot be overlap except 0');
	                                                    }else {
                                                            if(row.apnType == '3' && ids.length > 1 ) {
                                                                cb('Single number is from 0 to 4094');
                                                            //}else if(row.apnType == '2' & ids.length > 2) {
	                                                        //    cb('Allows up to 2 comma-separated numbers,range of each number is from 0 to 4094');
                                                            }else {
                                                                if(row.apnType == '2') {
                                                                    let tunnelVlans = form.apnConfig.filter(function(m){
                                                                            return m.apnOrder != row.apnOrder && ['2'].includes(m.apnType);
                                                                        }),
                                                                        tunnelIds = [],
                                                                        tunnelVlanIdRepeated = false;
        
                                                                    tunnelVlans.map(function(o){
                                                                        if(o.vlanId.length) {
                                                                            o.vlanId.split(',').map(function(m){
                                                                                tunnelIds.push(m||'0');
                                                                            });
                                                                        }
                                                                    });
        
                                                                    list.map(function(m){
                                                                        if(tunnelIds.includes(m)) tunnelVlanIdRepeated = true;
                                                                    })
        
                                                                    if(tunnelVlanIdRepeated) {
                                                                        cb('Tunnel mode Vlan ID cannot be overlap');
                                                                    }else {
                                                                        cb();
                                                                    }
                                                                }else {
                                                                    cb();
                                                                }
                                                            }
	                                                    }
		                                            }
												}else {
                                                    if(row.apnType == '3') {
                                                        if([null,''].includes(val)) {
                                                            cb();
                                                        }else {
                                                            cb('Single number is from 0 to 4094');
                                                        }
                                                    }else {
                                                        if(row.untaggedVlan - 1 >= 0) {
													        //cb('Allows up to 2 comma-separated numbers,range of each number is from 1 to 4094');
													        cb('Allows comma-separated numbers,range of each number is from 1 to 4094');
                                                        }else {
													        //cb('Allows up to 2 comma-separated numbers,range of each number is from 0 to 4094');
													        cb('Allows comma-separated numbers,range of each number is from 0 to 4094');
                                                        }
                                                    }
												}
		                                    }
		                                 }">
		                        		<el-input v-model="row.taggedVlan" @change="function(val){
                                            $refs.form.validateField('apnConfig.'+idx+'.untaggedVlan');

                                            if(row.apnType == '2') {
                                                var list = [];

                                                if(![null,undefined,''].includes(row.untaggedVlan)) {
                                                    row.untaggedVlan.split(',').map(function(m){
                                                        if(!list.includes(m)) list.push(m);
                                                    });
                                                }
                                                if(![null,undefined,''].includes(row.taggedVlan)) {
                                                    row.taggedVlan.split(',').map(function(m){
                                                        if(!list.includes(m)) list.push(m);
                                                    });
                                                }
                                                row.vlanId = list.join(',');
                                            }
                                        }"></el-input>
		                        	</el-form-item>
	                        	</div>
                        	</div>
                        </div>
                    </div>
                </div>
            </div>
            <el-form-item v-show="false" prop="apnConfig">
                <el-input v-model="form.apnConfig"></el-input>
            </el-form-item>
        </div>

        <div class="group-title not-extend margin-left-px sub-title">
            <span class="title-icon"></span>
            <span class="title-text split-title" style="padding-top: 0;display: flex;width: 100%;">
                Advance <i :class="arrowCls" @click="isDown = !isDown"></i>
            </span>
        </div>
        <div v-show="!isDown" class="flex-form flex-wrap" style="padding: 15px 0px 0px 80px;position: relative;">
            <div style="position:absolute;left: 30px; top: 23px;font-weight: bold;" v-show="form.l2TunnelEnable == 'true'">ENB</div>
            <div style="position:absolute;left: 30px; top: 80px;font-weight: bold;">CPE</div>
            <el-form-item v-show="false" label="L2 Tunnel Enable" class="third" label-width="150" prop="l2TunnelEnable">
                <el-switch v-model="form.l2TunnelEnable" active-value="true" inactive-value="false" style="zoom: 0.8"></el-switch>
            </el-form-item>
            <el-form-item v-show="form.l2TunnelEnable == 'true'" label="L2 Server IP" class="third" prop="l2ServerIp" label-width="150">
                <el-input v-model="form.l2ServerIp"></el-input>
            </el-form-item>
            <el-form-item v-if="form.l2TunnelEnable != 'true'" label=" " class="third" label-width="150">
                <el-input style="opacity: 0;"></el-input>
            </el-form-item>
            <el-form-item v-show="false" label="LGW Mode" class="third" label-width="120">
                <el-select v-model="form.lgwMode">
                    <el-option label="NAT" value="0"></el-option>
                    <el-option label="Bridge" value="2"></el-option>
                </el-select>
            </el-form-item>
            <el-form-item label="" class="third readonly-cls" prop="lgwIpAddress" label-width="120">
                <el-input v-if="false" v-model="form.lgwIpAddress" :readonly="true"></el-input>
            </el-form-item>
            <el-form-item label="" class="third readonly-cls" label-width="120">
                
            </el-form-item>
            <el-form-item label="L2 Destination IP" class="third readonly-cls" prop="destinationIp" label-width="150">
                <el-input v-model="form.destinationIp"></el-input>
            </el-form-item>
            <el-form-item v-show="false" label="Operation Mode">
                <el-select v-model="form.l2TunnelMode" disabled>
                    <el-option label="NAT" value="0"></el-option>
                    <el-option label="Tunnel" value="2"></el-option>
                    <el-option label="Bridge" value="3"></el-option>
                </el-select>
            </el-form-item>
        </div>
    </el-form>

    <el-dialog ref="dl" title="<%=rb.getString("QueRen")%>" :visible.sync="dlShow" :append-to-body="true" width="500">
        <div>
            The changes will take effect on the 
            <el-checkbox v-model="enbChecked" v-if="enbShow">HaloBs</el-checkbox>
            <span v-if="enbShow && cpeShow"> and </span>
            <el-checkbox v-model="cpeChecked" v-if="cpeShow">CPEs</el-checkbox>
            <div>
                <el-checkbox v-model="rebootChecked"><%=rb.getString("SheZhiHouChongQi")%></el-checkbox>
            </div>
        </div>
        <span slot="footer">
            <el-button type="primary" @click="save"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="dlShow=false"><%=rb.getString("QuXiao")%></el-button>
        </span>
    </el-dialog>
</div>

<script>
    new Vue({
        el: '#apn_setting_ctn',
        data() {
            var vm = this,
                validateIp = function(rule,value,cb) {
                    if(value && isValidIP(value)) {
                        cb();
                    }else if(value) {
                        cb('<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>')
                    }else {
                        cb();
                    }
                },
                validConfig = function(rule,value,cb) {
                    var enableItems = value||[],
                        isBearTypeRight = false,
                        bearTypes = enableItems.map(function(item){
                            return item.bearType;
                        });
                    
                    
                    if(bearTypes.includes('1')) isBearTypeRight = true;

                    if(isBearTypeRight) {
                        cb();
                    }else {
                        cb('<%=rb.getString("XuanZeBearType")%>');
                    }
                },
                validateMtu = function(rule,val,cb) {
                	if(val !== '' && (val -700 < 0 || val - 1600 > 0 || isNaN(val))) {
                		cb('Range: 700 - 1600');
                	}else {
                		cb();
                	}
                };

            return {
                rebootChecked: true,

                form: {
                    destinationIp: '',
                    l2TunnelMode: '',
                    mtu: '',
                    cpeMtu: '',
					
                    automaticEnable: 'true',
                    l2TunnelEnable: '',
                    l2ServerIp: '',
                    lgwMode: '',
                    lgwIpAddress: '',
                    apnConfig: []
                },
                rules: {
                    destinationIp: [{validator: validateIp}],
                    lgwIpAddress: [{validator: validateIp}],
                    mtu: [{validator: validateMtu}],
                    cpeMtu: [{validator: validateMtu}],
                },
                isDown: true,
                dlShow: false,
                enbChecked: false,
                cpeChecked: false,
                enbShow: false,
                cpeShow: false
            };
        },
        computed: {
        	isBridge() {
        		var vm = this,
        			bool = false;
        		
        		if(vm.form.apnConfig) {
        			vm.form.apnConfig.map(function(item) {
        				if(item.apnType == '2' || item.apnType == '3') {
        					bool = true;
        				}
        			});
        		}
        		
        		return bool;
        	},
            halfCount() {
                var vm = this;

                return Math.round(vm.form.apnConfig.length/2);
            },
            arrowCls() {
                var vm = this;

                return {
                    'el-icon': true,
                    'el-icon-down': vm.isDown,
                    'el-icon-up': !vm.isDown
                }
            }
        },
        methods: {
        	init() {
        		var vm = this;
        		
        		vm.initEnbAPN();
        	},
        	initEnbAPN() {
        		var vm = this;
        		
        		axios.post('${ctx}/epc/apnl2config/queryApnL2Config.action').then(function(res){
        			var data = res.data || {};
        			
                    if(data['apnConfig']) {
                        data['apnConfig'].map(function(itm){
                            if(itm.defaultApn == '1' && itm.vlanId == '0') itm.vlanId = '';
                        });
                    }

        			['cpeMtu','mtu','automaticEnable','lgwMode','l2TunnelMode','destinationIp','l2ServerIp', 'lgwIpAddress', 'apnConfig', 'l2TunnelEnable'].map(function(code){
        				if(data[code] !== undefined) {
        					vm.form[code] = data[code];
        				}
        			});
        			
        			//vm.form.apnConfig = [{apnOrder:1},{apnOrder:2},{apnOrder:3}]
        			
        			initForm(vm.$refs.form);
        		})
        	},
            getLayer(mode, apnType) {
                if(mode == '0') {
                    return 'L3';
                }else if(['2','3'].includes(mode)){
                    if(apnType == '1') {
                        return 'L3'
                    }else if(['0'].includes(apnType)){
                        return 'L2'
                    }else {
                        return '';
                    }
                }else {
                    return '';
                }
                
            },
            linkTunnel() {
                var vm = this,
                    hasEnable = false,
                    isBridge = false;

                vm.form.apnConfig.map(function(item){
                    if(item.apnType == '2') hasEnable = true;
                    
                    if(item.apnType == '2' || item.apnType == '3') {
                    	isBridge = true;
                    }
                });

                if(isBridge) {
                	vm.form.lgwMode = '2';
                }else {
                	vm.form.lgwMode = '0';
                }
                
                if(hasEnable) {
                	vm.form.l2TunnelMode = '2';
                }else if(isBridge) {
                	vm.form.l2TunnelMode = '3';
                }else {
                	vm.form.l2TunnelMode = '0';
                }
                
                var newVal = hasEnable?'true':'false';
                if(newVal != vm.form.l2TunnelEnable) {
                    vm.form.l2TunnelEnable = newVal;
                    vm.$message({
                        message: 'eNB L2 Tunnel will be changed to ' + (newVal=='true'?'enable':'disable'),
                        type: 'warning'
                    })
                }
            },
            propHasDotted(prop) {
                
                return prop.indexOf('.') >= 0;
            },
            //form表单是否发生改变
            isFormChanged(form){
                var vm = this,
                    isChanged = false,
                    changedProps = [];

                if(form.isReinited){
                    form.fields.map(function(field){
                        if(Array.isArray(field.fieldValue)){
                            var vList = field.fieldValue.map(function(item){return item});
                            var oList = (field.reinitialValue||[]).map(function(item){return item});
                            var val = JSON.stringify(vList.sort());
                            var orVal = JSON.stringify(oList.sort());
                            if(val != orVal) {
                                isChanged = true;
                                changedProps.push(field.prop);
                            }
                        }else{
                            if(isNull(field.fieldValue) && isNull(field.reinitialValue)){
                                
                            }else if(field.fieldValue !== field.reinitialValue && !vm.propHasDotted(field.prop)) {
                                isChanged = true;
                                changedProps.push(field.prop);
                            }
                        }
                    })
                }else{
                    form.fields.map(function(field){
                        if(Array.isArray(field.fieldValue)){
                            var vList = field.fieldValue.map(function(item){return item});
                            var oList = (field.initialValue||[]).map(function(item){return item});
                            var val = JSON.stringify(vList.sort()); 
                            var orVal = JSON.stringify(oList.sort());
                            if(val != orVal) {
                                isChanged = true;
                                changedProps.push(field.prop);
                            }
                        }else{
                            if(isNull(field.fieldValue) && isNull(field.initialValue)){
                                
                            }else if(field.fieldValue !== field.initialValue && !vm.propHasDotted(field.prop)) {
                                isChanged = true;
                                changedProps.push(field.prop);
                            }
                        }
                    })
                }

                return {
                    status: isChanged,
                    props: changedProps
                };

                function isNull(val){
                    if(val==undefined || val == null || val =="") return true;
                    else return false;
                }
            },
            beforeSave() {
                var vm = this,
                    result = vm.isFormChanged(vm.$refs.form),
                    isEnbChanged = false;
                    isCpeChanged = false;
                
                vm.$refs.form.validate(function(r){
                    if(r) {
                        if(result.status) {
                            result.props.map(function(prop){
                                if(['mtu','automaticEnable','lgwMode','l2TunnelEnable','l2ServerIp','lgwIpAddress','apnConfig'].includes(prop)) {
                                    isEnbChanged = true;
                                }
                                if(['cpeMtu','automaticEnable','lgwMode','l2TunnelMode','destinationIp','apnConfig'].includes(prop)) {
                                    isCpeChanged = true;
                                }
                            });

                            vm.enbChecked = isEnbChanged;
                            vm.cpeChecked = isCpeChanged;
                            vm.enbShow = isEnbChanged;
                            vm.cpeShow = isCpeChanged;
                            vm.dlShow = true;
                            vm.rebootChecked = true;
                        }else {
                            vm.$message('<%=rb.getString("WuCanShuBianHua")%>');
                        }
                    }
                });
            },
            save() {
                var vm = this,
                    url = '${ctx}/epc/apnl2config/updateApnL2Config.action',
                    params = JSON.parse(JSON.stringify(vm.form));
                 
                params.apnConfig.map(function(itm){
                    if(itm.defaultApn == '1' && itm.vlanId === '') itm.vlanId = '0';

                    if(['2','3'].includes(itm.apnType) && ['','0',null].includes(itm.untaggedVlan) && ['','0',null].includes(itm.taggedVlan)) {
                        itm.untaggedVlan = '0';
                        itm.taggedVlan = '0';
                    }
                });

                params.apnConfig = JSON.stringify(params.apnConfig);

                Object.assign(params,{
                    sendEnb: vm.enbChecked,
                    sendCpe: vm.cpeChecked,
                    isReboot: vm.rebootChecked? '1':'0'
                });
				
                vm.$refs.form.validate(function(r){
                    if(r) {
                        axios.post(url, stringify(params)).then(function(res){
                            var data = res.data;

                            if(data['success'] ) {
                            	eventBus.$message({
                                    message: '<%=rb.getString("ChengGong")%>',
                                    type: 'success'
                                });
                            	
                            	eventBus.$emit('hide-apn');
                                vm.dlShow = false;
                            }else {
                            	vm.$message({
                                    message: data.message,
                                    type: 'error'
                                });
                            }
                        });
                    }
                });
            }
        },
        mounted() {
        	this.init();

    	    eventBus.$off('save-apn').$on('save-apn', this.beforeSave);
        }
    })
</script>