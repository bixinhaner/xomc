<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
#networkPanel .el-form-item__label {
    margin-left: 0 !important;
}
#networkPanel .wanPageInfoWarp {
    background: #F6F7FB;
    border-radius: 4px;
    box-sizing: border-box;
    border: 1px solid #D5DCEC;
    margin-right: 40px;
    padding: 10px 20px;
    margin-bottom: 20px;
}
#networkPanel .wanPageInfoBoxCls {
    display: flex;
    flex-wrap: wrap;
}
#networkPanel .wanPageInfoBoxCls .cell-item-cls {
    width: 20%;
    min-width: 200px;
    min-height: 56px;
    font-size: 12px;
    color: rgba(0, 0, 0, 0.8);
}
#networkPanel .wanPageInfoBoxCls>div:before {
   content: attr(label);
   display: block;
   color: #7A7992;
   font-size: 12px;
   margin-bottom: 5px;
}
#networkPanel .enbIpsecEnableItem {
    display: flex !important;
}
#networkPanel .enbIpsecEnableItem .el-form-item__content {
    margin: 3px 0 0 30px;
}
#networkPanel .el-collapse-item__header {
    min-width: 100%;
}
#networkPanel .el-form-item__error {
    width: auto;
}
</style>
<div id='networkPanel' :class="addIpsecLoading ? 'loading' : ''" style='position:relative;background: #fff;height: 100%;width: 100%;'>
    <div v-show="!showIpsecAdd">
        <el-form ref="networkForm" :model="networkForm" :rules="networkRules" 
        label-position="top" style='width:100%;height:100%;' inline>
            <el-collapse v-model="activeNames">
                <el-collapse-item name="wan">
                    <template slot='title'>
                        <p style="display:inline-block;margin-left:40px;">
                            <span class="title-icon" style="vertical-align:sub"></span>
                            <span style="font-size:14px;font-weight:bold">WAN</span>
                        </p>
                    </template>
                    <!--['Intel_CR_CA','Intel_CR_SC','Intel_CR_TC','Intel_CR_DC','MLN_SC','MLN_CA','MLN_DC'].includes(platformType) 及 platformType.indexOf('BAIBLX')>=0 可配置这些-->
                    <div v-if="['Intel_CR_CA','Intel_CR_SC','Intel_CR_TC','Intel_CR_DC','Intel_CR','MLQ','MLN_SC','MLN_CA','MLN_DC'].includes(platformType) || platformType.indexOf('BLX')>=0 || platformType.indexOf('BAIBLQ')>=0">
                        <el-form-item label="Connect Type" prop="connectType" class='validate-item'>
                            <el-select v-model="networkForm.connectType" :disabled="isBaiblx_QRTB || isBaiblx_BLQ">
                                <el-option v-if="['Intel_CR_CA','Intel_CR_SC','Intel_CR_TC','Intel_CR_DC','Intel_CR','MLN_SC','MLN_CA','MLN_DC','MLN'].includes(platformType)" label='Auto' value='auto'></el-option>
                                <el-option label='Fiber' value='fiber'></el-option>
                                <el-option label='Copper' value='copper'></el-option>
                            </el-select>
                        </el-form-item>
                        <!--4860展示-->
                        <el-form-item label="Link Speed Negotiated" prop="linkSpeedNegotiated" class='validate-item' v-if="['Intel_CR_CA','Intel_CR_SC','Intel_CR_TC','Intel_CR_DC','Intel_CR','MLN_SC','MLN_CA','MLN_DC','MLN'].includes(platformType)">
                            <el-select v-model="networkForm.linkSpeedNegotiated">
                                <el-option label='Auto Negotiation' value='0'></el-option>
                                <el-option label='1000M' value='2'></el-option>
                            </el-select>
                        </el-form-item>
                        <!--Connect Type: Fiber,显示 SPF 信息-->
                        <div class='wanPageInfoWarp' v-if='networkForm.connectType == "fiber" && false'>
                            <p class='item-title-cls commonGeneral12' style='margin-bottom: 10px;'>SFP Information</p>
                            <div class="wanPageInfoBoxCls">
                                <div class="cell-item-cls" label="Identifier">SFP</div>
                                <div class="cell-item-cls" label="Connector">LC</div>
                                <div class="cell-item-cls" label="Transceiver">SingleMode</div>
                                <div class="cell-item-cls" label="Encoding">8B108</div>
                                <div class="cell-item-cls" label="Length">10KM</div>
                                <div class="cell-item-cls" label="Vendor Name">WTD</div>
                                <div class="cell-item-cls" label="Vendor PN">RTXM191-404</div>
                                <div class="cell-item-cls" label="Wavelength">1310NM</div>
                                <div class="cell-item-cls" label="SEP Options">TX--DISABLE TX_FAULT RX_LOS supported</div>
                                <div class="cell-item-cls" label="Bit Rate">1250Mbps</div>
                            </div>
                        </div>
                        <!--WAN Config-->
                        <div style='margin-right: 40px;'>
                            <div class='item-title-cls commonFlex' style='font-weight: unset; justify-content: space-between; margin-bottom: 10px;'>
                                <div class='commonFlex'>
                                    <p class='commonText14'>WAN Config</p>
                                    <p class='commonNotes12' style='margin-left: 10px;'>No more than {{wanConfigNumber}} configurations</p>
                                    <span style='color: #FA5151; margin-left: 15px;'><%=rb.getString("SheZhiHouChongQi")%></span>
                                </div>
                            </div>
                            <div v-show='false'>
                                <el-form-item prop="ipAccessMode1"><el-input v-model="networkForm.ipAccessMode1"></el-input></el-form-item>
                                <el-form-item prop="ipAddress1"><el-input v-model="networkForm.ipAddress1"></el-input></el-form-item>
                                <el-form-item prop="netmask1"><el-input v-model="networkForm.netmask1"></el-input></el-form-item>
                                <el-form-item prop="gateway1"><el-input v-model="networkForm.gateway1"></el-input></el-form-item>
                                <el-form-item prop="ipv6IpAddress1"><el-input v-model="networkForm.ipv6IpAddress1"></el-input></el-form-item>
                                <el-form-item prop="prefix1"><el-input v-model="networkForm.prefix1"></el-input></el-form-item>
                                <el-form-item prop="ipv6Gateway1"><el-input v-model="networkForm.ipv6Gateway1"></el-input></el-form-item>
                                <el-form-item prop="vlanId1"><el-input v-model="networkForm.vlanId1"></el-input></el-form-item>
                                <!--<el-form-item prop="effectEnable1"><el-input v-model="networkForm.effectEnable1"></el-input></el-form-item>-->
                                <el-form-item prop="option601"><el-input v-model="networkForm.option601"></el-input></el-form-item>

                                <el-form-item prop="ipAccessMode2"><el-input v-model="networkForm.ipAccessMode2"></el-input></el-form-item>
                                <el-form-item prop="ipAddress2"><el-input v-model="networkForm.ipAddress2"></el-input></el-form-item>
                                <el-form-item prop="netmask2"><el-input v-model="networkForm.netmask2"></el-input></el-form-item>
                                <el-form-item prop="gateway2"><el-input v-model="networkForm.gateway2"></el-input></el-form-item>
                                <el-form-item prop="prefix2"><el-input v-model="networkForm.prefix2"></el-input></el-form-item>
                                <el-form-item prop="ipv6IpAddress2"><el-input v-model="networkForm.ipv6IpAddress2"></el-input></el-form-item>
                                <el-form-item prop="ipv6Gateway2"><el-input v-model="networkForm.ipv6Gateway2"></el-input></el-form-item>
                                <el-form-item prop="vlanId2"><el-input v-model="networkForm.vlanId2"></el-input></el-form-item>
                                <el-form-item prop="effectEnable2"><el-input v-model="networkForm.effectEnable2"></el-input></el-form-item>
                                <el-form-item prop="option602"><el-input v-model="networkForm.option602"></el-input></el-form-item>

                                <el-form-item prop="ipAccessMode3"><el-input v-model="networkForm.ipAccessMode3"></el-input> </el-form-item>
                                <el-form-item prop="ipAddress3"> <el-input v-model="networkForm.ipAddress3"></el-input> </el-form-item>
                                <el-form-item prop="netmask3"> <el-input v-model="networkForm.netmask3"></el-input> </el-form-item>
                                <el-form-item prop="gateway3"><el-input v-model="networkForm.gateway3"></el-input></el-form-item>
                                <el-form-item prop="prefix3"><el-input v-model="networkForm.prefix3"></el-input></el-form-item>
                                <el-form-item prop="ipv6IpAddress3"><el-input v-model="networkForm.ipv6IpAddress3"></el-input></el-form-item>
                                <el-form-item prop="ipv6Gateway3"><el-input v-model="networkForm.ipv6Gateway3"></el-input></el-form-item>
                                <el-form-item prop="vlanId3"><el-input v-model="networkForm.vlanId3"></el-input></el-form-item>
                                <el-form-item prop="effectEnable3"><el-input v-model="networkForm.effectEnable3"></el-input></el-form-item>
                                <el-form-item prop="option603"><el-input v-model="networkForm.option603"></el-input></el-form-item>

                                <el-form-item prop="ipAccessMode4"><el-input v-model="networkForm.ipAccessMode4"></el-input></el-form-item>
                                <el-form-item prop="ipAddress4"><el-input v-model="networkForm.ipAddress4"></el-input></el-form-item>
                                <el-form-item prop="netmask4"><el-input v-model="networkForm.netmask4"></el-input></el-form-item>
                                <el-form-item prop="gateway4"><el-input v-model="networkForm.gateway4"></el-input></el-form-item>
                                <el-form-item prop="prefix4"><el-input v-model="networkForm.prefix4"></el-input></el-form-item>
                                <el-form-item prop="ipv6IpAddress4"><el-input v-model="networkForm.ipv6IpAddress4"></el-input></el-form-item>
                                <el-form-item prop="ipv6Gateway4"><el-input v-model="networkForm.ipv6Gateway4"></el-input></el-form-item>
                                <el-form-item prop="vlanId4"><el-input v-model="networkForm.vlanId4"></el-input></el-form-item>
                                <el-form-item prop="effectEnable4"><el-input v-model="networkForm.effectEnable4"></el-input></el-form-item>
                                <el-form-item prop="option604"><el-input v-model="networkForm.option604"></el-input></el-form-item>
                                <!--12 组参数  4860 模式，只配置四组-->
                                <div v-if="!['Intel_CR_CA','Intel_CR_SC','Intel_CR_TC','Intel_CR_DC','Intel_CR','MLN_SC','MLN_CA','MLN_DC','MLN'].includes(platformType)">
                                    <el-form-item prop="ipAccessMode5"><el-input v-model="networkForm.ipAccessMode5"></el-input></el-form-item>
                                    <el-form-item prop="ipAddress5"><el-input v-model="networkForm.ipAddress5"></el-input></el-form-item>
                                    <el-form-item prop="netmask5"><el-input v-model="networkForm.netmask5"></el-input></el-form-item>
                                    <el-form-item prop="gateway5"><el-input v-model="networkForm.gateway5"></el-input></el-form-item>
                                    <el-form-item prop="prefix5"><el-input v-model="networkForm.prefix5"></el-input></el-form-item>
                                    <el-form-item prop="ipv6IpAddress5"><el-input v-model="networkForm.ipv6IpAddress5"></el-input></el-form-item>
                                    <el-form-item prop="ipv6Gateway5"><el-input v-model="networkForm.ipv6Gateway5"></el-input></el-form-item>
                                    <el-form-item prop="vlanId5"><el-input v-model="networkForm.vlanId5"></el-input></el-form-item>
                                    <el-form-item prop="effectEnable5"><el-input v-model="networkForm.effectEnable5"></el-input></el-form-item>
                                    <el-form-item prop="option605"><el-input v-model="networkForm.option605"></el-input></el-form-item>

                                    <el-form-item prop="ipAccessMode6"><el-input v-model="networkForm.ipAccessMode6"></el-input></el-form-item>
                                    <el-form-item prop="ipAddress6"><el-input v-model="networkForm.ipAddress6"></el-input></el-form-item>
                                    <el-form-item prop="netmask6"><el-input v-model="networkForm.netmask6"></el-input></el-form-item>
                                    <el-form-item prop="gateway6"><el-input v-model="networkForm.gateway6"></el-input></el-form-item>
                                    <el-form-item prop="prefix6"><el-input v-model="networkForm.prefix6"></el-input></el-form-item>
                                    <el-form-item prop="ipv6IpAddress6"><el-input v-model="networkForm.ipv6IpAddress6"></el-input></el-form-item>
                                    <el-form-item prop="ipv6Gateway6"><el-input v-model="networkForm.ipv6Gateway6"></el-input></el-form-item>
                                    <el-form-item prop="vlanId6"><el-input v-model="networkForm.vlanId6"></el-input></el-form-item>
                                    <el-form-item prop="effectEnable6"><el-input v-model="networkForm.effectEnable6"></el-input></el-form-item>
                                    <el-form-item prop="option606"><el-input v-model="networkForm.option606"></el-input></el-form-item>

                                    <el-form-item prop="ipAccessMode7"><el-input v-model="networkForm.ipAccessMode7"></el-input></el-form-item>
                                    <el-form-item prop="ipAddress7"><el-input v-model="networkForm.ipAddress7"></el-input></el-form-item>
                                    <el-form-item prop="netmask7"><el-input v-model="networkForm.netmask7"></el-input></el-form-item>
                                    <el-form-item prop="gateway7"><el-input v-model="networkForm.gateway7"></el-input></el-form-item>
                                    <el-form-item prop="prefix7"><el-input v-model="networkForm.prefix7"></el-input></el-form-item>
                                    <el-form-item prop="ipv6IpAddress7"><el-input v-model="networkForm.ipv6IpAddress7"></el-input></el-form-item>
                                    <el-form-item prop="ipv6Gateway7"><el-input v-model="networkForm.ipv6Gateway7"></el-input></el-form-item>
                                    <el-form-item prop="vlanId7"><el-input v-model="networkForm.vlanId7"></el-input></el-form-item>
                                    <el-form-item prop="effectEnable7"><el-input v-model="networkForm.effectEnable7"></el-input></el-form-item>
                                    <el-form-item prop="option607"><el-input v-model="networkForm.option607"></el-input></el-form-item>

                                    <el-form-item prop="ipAccessMode8"><el-input v-model="networkForm.ipAccessMode8"></el-input></el-form-item>
                                    <el-form-item prop="ipAddress8"><el-input v-model="networkForm.ipAddress8"></el-input></el-form-item>
                                    <el-form-item prop="netmask8"><el-input v-model="networkForm.netmask8"></el-input></el-form-item>
                                    <el-form-item prop="gateway8"><el-input v-model="networkForm.gateway8"></el-input></el-form-item>
                                    <el-form-item prop="prefix8"><el-input v-model="networkForm.prefix8"></el-input></el-form-item>
                                    <el-form-item prop="ipv6IpAddress8"><el-input v-model="networkForm.ipv6IpAddress8"></el-input></el-form-item>
                                    <el-form-item prop="ipv6Gateway8"><el-input v-model="networkForm.ipv6Gateway8"></el-input></el-form-item>
                                    <el-form-item prop="vlanId8"><el-input v-model="networkForm.vlanId8"></el-input></el-form-item>
                                    <el-form-item prop="effectEnable8"><el-input v-model="networkForm.effectEnable8"></el-input></el-form-item>
                                    <el-form-item prop="option608"><el-input v-model="networkForm.option608"></el-input></el-form-item>

                                    <el-form-item prop="ipAccessMode9"><el-input v-model="networkForm.ipAccessMode9"></el-input></el-form-item>
                                    <el-form-item prop="ipAddress9"><el-input v-model="networkForm.ipAddress9"></el-input></el-form-item>
                                    <el-form-item prop="netmask9"><el-input v-model="networkForm.netmask9"></el-input></el-form-item>
                                    <el-form-item prop="gateway9"><el-input v-model="networkForm.gateway9"></el-input></el-form-item>
                                    <el-form-item prop="prefix9"><el-input v-model="networkForm.prefix9"></el-input></el-form-item>
                                    <el-form-item prop="ipv6IpAddress9"><el-input v-model="networkForm.ipv6IpAddress9"></el-input></el-form-item>
                                    <el-form-item prop="ipv6Gateway9"><el-input v-model="networkForm.ipv6Gateway9"></el-input></el-form-item>
                                    <el-form-item prop="vlanId9"><el-input v-model="networkForm.vlanId9"></el-input></el-form-item>
                                    <el-form-item prop="effectEnable9"><el-input v-model="networkForm.effectEnable9"></el-input></el-form-item>
                                    <el-form-item prop="option609"><el-input v-model="networkForm.option609"></el-input></el-form-item>

                                    <el-form-item prop="ipAccessMode10"><el-input v-model="networkForm.ipAccessMode10"></el-input></el-form-item>
                                    <el-form-item prop="ipAddress10"><el-input v-model="networkForm.ipAddress10"></el-input></el-form-item>
                                    <el-form-item prop="netmask10"><el-input v-model="networkForm.netmask10"></el-input></el-form-item>
                                    <el-form-item prop="gateway10"><el-input v-model="networkForm.gateway10"></el-input></el-form-item>
                                    <el-form-item prop="ipv6IpAddress10"><el-input v-model="networkForm.ipv6IpAddress10"></el-input></el-form-item>
                                    <el-form-item prop="prefix10"><el-input v-model="networkForm.prefix10"></el-input></el-form-item>
                                    <el-form-item prop="ipv6Gateway10"><el-input v-model="networkForm.ipv6Gateway10"></el-input></el-form-item>
                                    <el-form-item prop="vlanId10"><el-input v-model="networkForm.vlanId10"></el-input></el-form-item>
                                    <el-form-item prop="effectEnable10"><el-input v-model="networkForm.effectEnable10"></el-input></el-form-item>
                                    <el-form-item prop="option6010"><el-input v-model="networkForm.option6010"></el-input></el-form-item>

                                    <el-form-item prop="ipAccessMode11"><el-input v-model="networkForm.ipAccessMode11"></el-input></el-form-item>
                                    <el-form-item prop="ipAddress11"><el-input v-model="networkForm.ipAddress11"></el-input></el-form-item>
                                    <el-form-item prop="netmask11"><el-input v-model="networkForm.netmask11"></el-input></el-form-item>
                                    <el-form-item prop="gateway11"><el-input v-model="networkForm.gateway11"></el-input></el-form-item>
                                    <el-form-item prop="prefix11"><el-input v-model="networkForm.prefix11"></el-input></el-form-item>
                                    <el-form-item prop="ipv6IpAddress11"><el-input v-model="networkForm.ipv6IpAddress11"></el-input></el-form-item>
                                    <el-form-item prop="ipv6Gateway11"><el-input v-model="networkForm.ipv6Gateway11"></el-input></el-form-item>
                                    <el-form-item prop="vlanId11"><el-input v-model="networkForm.vlanId11"></el-input></el-form-item>
                                    <el-form-item prop="effectEnable11"><el-input v-model="networkForm.effectEnable11"></el-input></el-form-item>
                                    <el-form-item prop="option6011"><el-input v-model="networkForm.option6011"></el-input></el-form-item>

                                    <el-form-item prop="ipAccessMode12"><el-input v-model="networkForm.ipAccessMode12"></el-input></el-form-item>
                                    <el-form-item prop="ipAddress12"><el-input v-model="networkForm.ipAddress12"></el-input></el-form-item>
                                    <el-form-item prop="netmask12"><el-input v-model="networkForm.netmask12"></el-input></el-form-item>
                                    <el-form-item prop="gateway12"><el-input v-model="networkForm.gateway12"></el-input></el-form-item>
                                    <el-form-item prop="prefix12"><el-input v-model="networkForm.prefix12"></el-input></el-form-item>
                                    <el-form-item prop="ipv6IpAddress12"><el-input v-model="networkForm.ipv6IpAddress12"></el-input></el-form-item>
                                    <el-form-item prop="ipv6Gateway12"><el-input v-model="networkForm.ipv6Gateway12"></el-input></el-form-item>
                                    <el-form-item prop="vlanId12"><el-input v-model="networkForm.vlanId12"></el-input></el-form-item>
                                    <el-form-item prop="effectEnable12"><el-input v-model="networkForm.effectEnable12"></el-input></el-form-item>
                                    <el-form-item prop="option6012"><el-input v-model="networkForm.option6012"></el-input></el-form-item>
                                </div>
                            </div>
                            <div class="cellTableBoxCls" style="padding-bottom:20px;">
                                <el-ctable
                                    ref="wanConfigTable"
                                    :rownumber="true"
                                    id="wanConfigTable"
                                    :data="wanConfigTableList"
                                    height="225px"
                                    :pagination="false"
                                    style="border:1px solid #E9E9E9;">
                                    <el-table-column label='<%=rb.getString("CaoZuo")%>' min-width="100">
                                        <template slot-scope="scope">
                                            <span class="el-icon el-icon-operation-edit" @click="addWANDialogOpen(scope.row)" style="margin-right:15px;"></span>
                                        </template>
                                    </el-table-column>
                                    <el-table-column label='<%=rb.getString("ShiFouShengXiao")%>' min-width="150" prop="effectEnable" show-overflow-tooltip>
                                        <template slot-scope="scope">
                                            <div v-if="scope.row.effectEnable == '0'"> <%=rb.getString("Guan")%> </div>
                                            <div v-else-if="scope.row.effectEnable == '1'"> <%=rb.getString("Kai")%> </div>
                                            <div v-else> - </div>
                                        </template>
                                    </el-table-column>
                                    <el-table-column label='Index' min-width="80" prop="index" show-overflow-tooltip></el-table-column>
                                    <el-table-column label='WAN Name' min-width="120" prop="wanName" show-overflow-tooltip></el-table-column>
                                    <el-table-column label='IP Access Mode' min-width="120" prop="ipAccessMode" show-overflow-tooltip>
                                        <template slot-scope="scope">
                                            <div v-if="scope.row.ipAccessMode == '0'"> DHCP </div>
                                            <div v-else-if="scope.row.ipAccessMode == '1'"> Static IP </div>
                                            <div v-else-if="scope.row.ipAccessMode == '3' && !['Intel_CR_CA','Intel_CR_SC','Intel_CR_TC','Intel_CR_DC','Intel_CR','MLN_SC','MLN_CA','MLN_DC','MLN'].includes(platformType)"> IPV6 DHCP </div>
                                            <div v-else-if="scope.row.ipAccessMode == '4' && !['Intel_CR_CA','Intel_CR_SC','Intel_CR_TC','Intel_CR_DC','Intel_CR','MLN_SC','MLN_CA','MLN_DC','MLN'].includes(platformType)"> IPV6 Static IP </div>
                                        </template>
                                    </el-table-column>
                                    <!--普通三组-->
                                    <el-table-column label='IP Address' min-width="140" prop="ipAddress" show-overflow-tooltip>
                                        <template slot-scope="scope">
                                            <div v-if="scope.row.ipAddress == '' || scope.row.ipAddress == '0.0.0.0' || scope.row.ipAccessMode != '1'"> - </div>                                                                            
                                            <div v-else>{{scope.row.ipAddress}}</div>
                                        </template>
                                    </el-table-column>
                                    <el-table-column  label='Netmask' min-width="120" prop="netmask" show-overflow-tooltip>
                                        <template slot-scope="scope">
                                            <div v-if="scope.row.netmask == '' || scope.row.netmask == '0.0.0.0' || scope.row.ipAccessMode != '1'"> - </div>                                         
                                            <div v-else>{{scope.row.netmask}}</div>
                                        </template>
                                    </el-table-column>
                                    <el-table-column label='Gateway' min-width="120" prop="gateway" show-overflow-tooltip>
                                        <template slot-scope="scope">
                                            <div v-if="scope.row.gateway == '' || scope.row.gateway == '0.0.0.0' || scope.row.ipAccessMode != '1'"> - </div>                                      
                                            <div v-else>{{scope.row.gateway}}</div>
                                        </template>
                                    </el-table-column>
                                    <!--ipv6 三组-->                        
                                    <el-table-column label='IPV6 IP Address' min-width="140" prop="ipv6IpAddress" show-overflow-tooltip v-if="!['Intel_CR_CA','Intel_CR_SC','Intel_CR_TC','Intel_CR_DC','Intel_CR','MLN_SC','MLN_CA','MLN_DC','MLN'].includes(platformType)">
                                        <template slot-scope="scope">
                                            <div v-if="scope.row.ipv6IpAddress == '' || scope.row.ipAccessMode != '4'"> - </div>                                       
                                            <div v-else>{{scope.row.ipv6IpAddress}}</div>
                                        </template>
                                    </el-table-column>
                                    <el-table-column label='Prefix' min-width="120" prop="prefix" show-overflow-tooltip v-if="!['Intel_CR_CA','Intel_CR_SC','Intel_CR_TC','Intel_CR_DC','Intel_CR','MLN_SC','MLN_CA','MLN_DC','MLN'].includes(platformType)">
                                        <template slot-scope="scope">
                                            <div v-if="scope.row.prefix == '' || scope.row.ipAccessMode != '4'"> - </div>                                       
                                            <div v-else>{{scope.row.prefix}}</div>
                                        </template>
                                    </el-table-column>
                                    <el-table-column label='IPV6 Gateway' min-width="120" prop="ipv6Gateway" show-overflow-tooltip v-if="!['Intel_CR_CA','Intel_CR_SC','Intel_CR_TC','Intel_CR_DC','Intel_CR','MLN_SC','MLN_CA','MLN_DC','MLN'].includes(platformType)">
                                        <template slot-scope="scope">
                                            <div v-if="scope.row.ipv6Gateway == '' || scope.row.ipAccessMode != '4'"> - </div>                                       
                                            <div v-else>{{scope.row.ipv6Gateway}}</div>
                                        </template>
                                    </el-table-column>

                                    <el-table-column label='VLAN ID' min-width="100" prop="vlanId" show-overflow-tooltip>
                                        <template slot-scope="scope">                                   
                                            <div>{{scope.row.vlanId}}</div>
                                        </template>
                                    </el-table-column>
                                </el-ctable>
                            </div>
                        </div>
                        <div v-if="!isMLQ">
                            <div class='item-title-cls commonFlex' style='font-weight: unset; margin-bottom: 0px;'>
                                <p class='commonText14'>DNS Config</p>
                                <p class='commonNotes12' style='margin-left: 10px;'>No more than 2 configurations</p>
                                <el-form-item label="" prop="dnsConfigEnable" class='enbIpsecEnableItem'>
                                    <el-switch v-model="networkForm.dnsConfigEnable" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#BDC1C6"></el-switch>
                                </el-form-item>
                            </div>
                            <el-form-item label="DNS Address1" prop="dnsAddress" style='margin-left: 16px !important;'>
                                <el-input v-model="networkForm.dnsAddress"></el-input>
                            </el-form-item>
                            <el-form-item label="DNS Address2" prop="dnsAddress2">
                                <el-input v-model="networkForm.dnsAddress2"></el-input>
                            </el-form-item>
                        </div>
                        <div v-if="!isMLQ">
                            <p class='item-title-cls '>Other Config</p>
                            <el-form-item v-if="!['Intel_CR_CA','Intel_CR_SC','Intel_CR_TC','Intel_CR_DC','Intel_CR','BLX','MLN_SC','MLN_CA','MLN_DC','MLN'].includes(platformType)" label="MTU" style='margin-bottom:16px; margin-left: 16px !important;' prop="mtu" class='validate-item'>
                                <el-input v-model="networkForm.mtu">
                                    <template slot="append">Range:700~1600</template>
                                </el-input>
                            </el-form-item>
                            <el-form-item v-if="!['Intel_CR_CA','Intel_CR_SC','Intel_CR_TC','Intel_CR_DC','Intel_CR','MLN_SC','MLN_CA','MLN_DC','MLN'].includes(platformType)" label="Quick Interface Binding" prop="quickInterfaceBinding" class='validate-item'>
                                <el-select v-model="networkForm.quickInterfaceBinding">
                                    <el-option v-if="!(isBaiblx_QRTB || isBaiblx_BLQ)" v-for="item in quickInterfaceBindingList" :key="item.value" :label="item.text" :value="item.value"></el-option>
                                    <el-option v-if="isBaiblx_QRTB || isBaiblx_BLQ" v-for="item in newQuickBandList" :key="item.value" :label="item.text" :value="item.value"></el-option>
                                </el-select>
                            </el-form-item>
                            <el-form-item label="Access LMT via Wan" prop="accessLMTViaWAN" class='validate-item' style='margin-left: 16px !important;'>
                                <el-select v-model="networkForm.accessLMTViaWAN">
                                    <el-option label='ON' value='1'></el-option>
                                    <el-option label='OFF' value='0'></el-option>
                                </el-select>
                            </el-form-item>
                            <el-form-item v-if="isBaiblx_QRTB || isBaiblx_BLQ" label="Slave Interface" prop="slaveInterface">
                                <el-select v-model="networkForm.slaveInterface">
                                    <el-option v-for="item in slaveInterfaceList" :key="item.value" :label="item.text" :value="item.value"></el-option>
                                </el-select>
                            </el-form-item>

                            <el-form-item v-if="isBaiblx_QRTB || isBaiblx_BLQ" label="Sub Machine Address" prop="subMachineAddress">
                                <el-input v-model="networkForm.subMachineAddress"></el-input>
                            </el-form-item>
                        </div>
                        <div v-if="!(isBaiblx_QRTB || isBaiblx_BLQ || isMLQ)">
                            <p class='item-title-cls '>LAN Config</p>
                            <el-form-item label="IP Address" prop="lanIpAddress" style='margin-bottom:0px; margin-left: 16px !important;' >
                                <el-input v-model="networkForm.lanIpAddress" :disabled="['Intel_CR_CA','Intel_CR_SC','Intel_CR_TC','Intel_CR_DC','Intel_CR','MLN_SC','MLN_CA','MLN_DC','MLN'].includes(platformType)"></el-input>
                            </el-form-item>
                            <el-form-item label="Subnet Mask" prop="subnetMask">
                                <el-input v-model="networkForm.subnetMask" :disabled="['Intel_CR_CA','Intel_CR_SC','Intel_CR_TC','Intel_CR_DC','Intel_CR','MLN_SC','MLN_CA','MLN_DC','MLN'].includes(platformType)"></el-input>
                            </el-form-item>
                        </div>
                    </div>
                    <div v-else>
                        <!--platformType.indexOf('436Q')>=0 || platformType.indexOf('MLQ')>=0  这三个条件中可配置此项-->
                        <el-form-item label="MTU" style='margin-bottom:16px; margin-left: 16px !important;' prop="mtu" class='validate-item'>
                            <el-input v-model="networkForm.mtu">
                                <template slot="append">Range:700~1600</template>
                            </el-input>
                        </el-form-item>
                    </div>
                </el-collapse-item>

                <!-- Static Routing -->
                <el-collapse-item name="staticRouter" v-if="['Intel_CR_CA','Intel_CR_SC','Intel_CR_TC','Intel_CR_DC','Intel_CR','MLN_SC','MLN_CA','MLN_DC','MLN'].includes(platformType) || platformType.indexOf('BLX')>=0">
                    <template slot='title'>
                        <p style="display:inline-block;margin-left:40px;">
                            <span class="title-icon" style="vertical-align:sub"></span>
                            <span style="font-size:14px;font-weight:bold">Static Routing</span>
                            <p v-if="false" class='commonNotes12' style='margin-left: 10px;display: inline;'>The maximum number of configuration is 12</p>
                        </p>
                    </template>
                    <div>
                        <div v-show='false'>
                            <el-form-item prop="routeEffective1"><el-input v-model="networkForm.routeEffective1"></el-input></el-form-item>
                            <el-form-item prop="routeNetwork1"><el-input v-model="networkForm.routeNetwork1"></el-input></el-form-item>
                            <el-form-item prop="routeNetmask1"><el-input v-model="networkForm.routeNetmask1"></el-input></el-form-item>
                            <el-form-item prop="routeGateway1"><el-input v-model="networkForm.routeGateway1"></el-input></el-form-item>
                            
                            <el-form-item prop="routeEffective2"><el-input v-model="networkForm.routeEffective2"></el-input></el-form-item>
                            <el-form-item prop="routeNetwork2"><el-input v-model="networkForm.routeNetwork2"></el-input></el-form-item>
                            <el-form-item prop="routeNetmask2"><el-input v-model="networkForm.routeNetmask2"></el-input></el-form-item>
                            <el-form-item prop="routeGateway2"><el-input v-model="networkForm.routeGateway2"></el-input></el-form-item>
                            
                            <el-form-item prop="routeEffective3"><el-input v-model="networkForm.routeEffective3"></el-input></el-form-item>
                            <el-form-item prop="routeNetwork3"><el-input v-model="networkForm.routeNetwork3"></el-input></el-form-item>
                            <el-form-item prop="routeNetmask3"><el-input v-model="networkForm.routeNetmask3"></el-input></el-form-item>
                            <el-form-item prop="routeGateway3"><el-input v-model="networkForm.routeGateway3"></el-input></el-form-item>
                            
                            <el-form-item prop="routeEffective4"><el-input v-model="networkForm.routeEffective4"></el-input></el-form-item>
                            <el-form-item prop="routeNetwork4"><el-input v-model="networkForm.routeNetwork4"></el-input></el-form-item>
                            <el-form-item prop="routeNetmask4"><el-input v-model="networkForm.routeNetmask4"></el-input></el-form-item>
                            <el-form-item prop="routeGateway4"><el-input v-model="networkForm.routeGateway4"></el-input></el-form-item>
                            
                            <el-form-item prop="routeEffective5"><el-input v-model="networkForm.routeEffective5"></el-input></el-form-item>
                            <el-form-item prop="routeNetwork5"><el-input v-model="networkForm.routeNetwork5"></el-input></el-form-item>
                            <el-form-item prop="routeNetmask5"><el-input v-model="networkForm.routeNetmask5"></el-input></el-form-item>
                            <el-form-item prop="routeGateway5"><el-input v-model="networkForm.routeGateway5"></el-input></el-form-item>
                            
                            <el-form-item prop="routeEffective6"><el-input v-model="networkForm.routeEffective6"></el-input></el-form-item>
                            <el-form-item prop="routeNetwork6"><el-input v-model="networkForm.routeNetwork6"></el-input></el-form-item>
                            <el-form-item prop="routeNetmask6"><el-input v-model="networkForm.routeNetmask6"></el-input></el-form-item>
                            <el-form-item prop="routeGateway6"><el-input v-model="networkForm.routeGateway6"></el-input></el-form-item>
                            
                            <el-form-item prop="routeEffective7"><el-input v-model="networkForm.routeEffective7"></el-input></el-form-item>
                            <el-form-item prop="routeNetwork7"><el-input v-model="networkForm.routeNetwork7"></el-input></el-form-item>
                            <el-form-item prop="routeNetmask7"><el-input v-model="networkForm.routeNetmask7"></el-input></el-form-item>
                            <el-form-item prop="routeGateway7"><el-input v-model="networkForm.routeGateway7"></el-input></el-form-item>
                            
                            <el-form-item prop="routeEffective8"><el-input v-model="networkForm.routeEffective8"></el-input></el-form-item>
                            <el-form-item prop="routeNetwork8"><el-input v-model="networkForm.routeNetwork8"></el-input></el-form-item>
                            <el-form-item prop="routeNetmask8"><el-input v-model="networkForm.routeNetmask8"></el-input></el-form-item>
                            <el-form-item prop="routeGateway8"><el-input v-model="networkForm.routeGateway8"></el-input></el-form-item>
                            
                            <el-form-item prop="routeEffective9"><el-input v-model="networkForm.routeEffective9"></el-input></el-form-item>
                            <el-form-item prop="routeNetwork9"><el-input v-model="networkForm.routeNetwork9"></el-input></el-form-item>
                            <el-form-item prop="routeNetmask9"><el-input v-model="networkForm.routeNetmask9"></el-input></el-form-item>
                            <el-form-item prop="routeGateway9"><el-input v-model="networkForm.routeGateway9"></el-input></el-form-item>
                            
                            <el-form-item prop="routeEffective10"><el-input v-model="networkForm.routeEffective10"></el-input></el-form-item>
                            <el-form-item prop="routeNetwork10"><el-input v-model="networkForm.routeNetwork10"></el-input></el-form-item>
                            <el-form-item prop="routeNetmask10"><el-input v-model="networkForm.routeNetmask10"></el-input></el-form-item>
                            <el-form-item prop="routeGateway10"><el-input v-model="networkForm.routeGateway10"></el-input></el-form-item>
                            
                            <el-form-item prop="routeEffective11"><el-input v-model="networkForm.routeEffective11"></el-input></el-form-item>
                            <el-form-item prop="routeNetwork11"><el-input v-model="networkForm.routeNetwork11"></el-input></el-form-item>
                            <el-form-item prop="routeNetmask11"><el-input v-model="networkForm.routeNetmask11"></el-input></el-form-item>
                            <el-form-item prop="routeGateway11"><el-input v-model="networkForm.routeGateway11"></el-input></el-form-item>
                            
                            <el-form-item prop="routeEffective12"><el-input v-model="networkForm.routeEffective12"></el-input></el-form-item>
                            <el-form-item prop="routeNetwork12"><el-input v-model="networkForm.routeNetwork12"></el-input></el-form-item>
                            <el-form-item prop="routeNetmask12"><el-input v-model="networkForm.routeNetmask12"></el-input></el-form-item>
                            <el-form-item prop="routeGateway12"><el-input v-model="networkForm.routeGateway12"></el-input></el-form-item>
                        </div>

                        <div style="height:300px;max-height:300px;width:95%;border:1px solid #F3F3F3;margin-left: 15px;">
                            <el-ctable ref="ctableRouter" :data="routerList" :pagination="false">
                                <el-table-column label='<%=rb.getString("CaoZuo")%>' width="100" prop="" class-name="no-text-tips">
                                    <template slot-scope="scope">
                                        <span class="el-icon el-icon-operation-edit" @click="routeModifyOpen(scope.row, scope.$index)" style="cursor: pointer;"></span>
                                    </template>
                                </el-table-column>
                                <el-table-column label='<%=rb.getString("ShiFouShengXiao")%>' min-width="100" prop="routeEffective">
                                    <template slot-scope="scope">
                                        <div v-if="scope.row.routeEffective == '0'"> <%=rb.getString("Guan")%> </div>
                                        <div v-else-if="scope.row.routeEffective == '1'"> <%=rb.getString("Kai")%> </div>
                                        <div v-else> - </div>
                                    </template>
                                </el-table-column>
                                <el-table-column label="Destination Network" prop="routeNetwork"></el-table-column>
                                <el-table-column label="Netmask" prop="routeNetmask"></el-table-column>
                                <el-table-column label='<%=rb.getString("TunnelWangGuan")%>' prop="routeGateway"></el-table-column>
                            </el-ctable>
                        </div>
                    </div>
                </el-collapse-item>

                <el-collapse-item name="ipsec">
                    <template slot='title'>
                        <p style="display:inline-block;margin-left:40px;">
                            <span class="title-icon" style="vertical-align:sub"></span>
                            <span style="font-size:14px;font-weight:bold">IPSec</span>
                        </p>
                    </template>
                    <div>
                        <p class='item-title-cls '>IPSec Setting</p>
                        <el-form-item label="Enable" prop="ipsecEnable" class='enbIpsecEnableItem' style='margin-left:15px !important;' >
                            <el-switch v-model="networkForm.ipsecEnable" active-value="true" inactive-value="false" active-color="#4D84FF" inactive-color="#BDC1C6"></el-switch>
                        </el-form-item>
                        <div style='margin-top:5px;width:95%;display:flex; margin-left: 15px;'>
                            <span style='font-size:12px;font-weight:bold;flex:1'>IPSec Tunnel List <span style='color: #FA5151; margin-left: 15px;'><%=rb.getString("SheZhiHouChongQi")%></span></span>
                            <span v-show="networkForm.ipsecList.length<5" @click='addIpsec' class='el-icon el-icon-circle-add' style='font-size: 16px;' v-show="addIpsecFlag"></span>
                        </div>
                        <div style="height:150px;max-height:300px;width:95%;border:1px solid #F3F3F3;margin-left: 15px;">
                            <el-ctable ref="ctableIpsec" :data="networkForm.ipsecList" :pagination="false">
                                <el-table-column label="Operations" width="100" prop="" class-name="no-text-tips">
                                    <template slot-scope="scope">
                                        <span class="el-icon el-icon-operation-edit" @click="editIpsec(scope.row)" style="cursor: pointer;"></span>
                                        <span v-show="scope.$index > 1" class="el-icon el-icon-operation-delete" @click="delIpsec(scope.row)" style="cursor: pointer;"></span>
                                    </template>
                                </el-table-column>
                                <el-table-column label="Enable" prop="configEnable" :formatter="ipsecFmt"></el-table-column>
                                <el-table-column label='Tunnel Name' prop="tunnelName">
                                    <template slot-scope="scope">
                                        <span v-if="scope.$index < 2">ipsectunnel{{scope.$index+1}}</span>
                                        <span v-else>{{scope.row.tunnelName}}</span>
                                    </template>
                                </el-table-column>
                                <el-table-column label='<%=rb.getString("TunnelWangGuan")%>' prop="gateway"></el-table-column>
                            </el-ctable>
                        </div>
                        <el-form-item v-show="false" prop="ipsecList" class="validate-item">
                            <el-input v-model="networkForm.ipsecList"></el-input>
                        </el-form-item>
                    </div>
                </el-collapse-item>
                <el-collapse-item name="lgw" v-if="['Intel_CR_CA','Intel_CR_SC','Intel_CR_TC','Intel_CR_DC','Intel_CR','MLN_SC','MLN_CA','MLN_DC','MLN'].includes(platformType)">
                    <template slot='title'>
                        <p style="display:inline-block;margin-left:40px;">
                            <span class="title-icon" style="vertical-align:sub"></span>
                            <span style="font-size:14px;font-weight:bold">LGW</span>
                        </p>
                    </template>
                    <div>
                        <el-form-item label='<%=rb.getString("SheZhiKaiGuan")%>' prop="lgwEnable" class='validate-item'>
                            <el-switch v-model="networkForm.lgwEnable" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#BDC1C6"></el-switch>
                        </el-form-item>
                        <template v-if='networkForm.lgwEnable == "1"'>
                            <el-form-item label='<%=rb.getString("LicenseMoShi")%>' prop="lgwMode" class='validate-item'>
                                <el-select v-model="networkForm.lgwMode">
                                    <el-option label='NAT' value='0'></el-option>
                                    <el-option label='Router' value='1'></el-option>
                                    <el-option label='Bridge' value='2'></el-option>
                                </el-select>
                            </el-form-item>
                            <el-form-item label='<%=rb.getString("lgwConfigDiZhiChi")%>' prop="lgwIpPool" v-if='networkForm.lgwMode != "2"'>
                                <el-input v-model="networkForm.lgwIpPool" @change="lgwValidateStaticIP"></el-input>
                            </el-form-item>
                            <!--输入框 or 动态下拉框-->
                            <el-form-item label='<%=rb.getString("lgwConfigDiZhiChiYanMa")%>' prop="lgwIpPoolNetmask" v-if='networkForm.lgwMode != "2"'>
                                <el-select v-model="networkForm.lgwIpPoolNetmask" @change="lgwValidateStaticIP">
                                    <el-option label='255.255.255.0' value='255.255.255.0'></el-option>
                                    <el-option label='255.255.255.128' value='255.255.255.128'></el-option>
                                    <el-option label='255.255.255.192' value='255.255.255.192'></el-option>
                                    <el-option label='255.255.255.224' value='255.255.255.224'></el-option>
                                    <el-option label='255.255.255.240' value='255.255.255.240'></el-option>
                                    <el-option label='255.255.255.248' value='255.255.255.248'></el-option>
                                </el-select>
                            </el-form-item>
                            <el-form-item label='<%=rb.getString("lgwJingTaiDiZhiKaiGuan")%>' prop="lgwIpEnable" v-if='networkForm.lgwMode == "1"'>
                                <el-switch v-model="networkForm.lgwIpEnable" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#BDC1C6"></el-switch>
                            </el-form-item>
                            <template v-if='networkForm.lgwMode == "1" && networkForm.lgwIpEnable == "1"'>
                                <el-form-item label='<%=rb.getString("lgwQiShiDiZhi")%>' prop="lgwFirstIp">
                                    <el-input v-model="networkForm.lgwFirstIp" @change="lgwValidateStaticIP"></el-input>
                                </el-form-item>
                                <el-form-item label='<%=rb.getString("lgwJieShuDiZhi")%>' prop="lgwLastIp">
                                    <el-input v-model="networkForm.lgwLastIp" @change="lgwValidateStaticIP"></el-input>
                                </el-form-item>
                                <el-form-item label='<%=rb.getString("lgwImsiIPBangDingFanWei")%>' style='margin-bottom:0px;position:relative;width:100%' class="validate-item" :class="lgwBindCls">
                                    <el-input v-model='lgwBindImsi' style='width:200px;'></el-input>&nbsp;&nbsp;—&nbsp;&nbsp;<el-input v-model='lgwBindIp' style='width:200px;'></el-input>
                                    <span @click='lgwAddBind' class='form-bt el-icon el-icon-plus' style='vertical-align:middle' v-show="networkForm.lgwIpEnable"></span>
                                    <span class='item-tip'><%=rb.getString("IMSIIPFanWeiTiShi")%></span>
                                </el-form-item>
                                <div style='overflow:auto'>
                                    <el-form-item class='suffixItem' v-for='(domain,index) in lgwBindGroup' style='line-height:16px;width:auto;'>
                                        <div class='form-suffix' style='width:auto;'>
                                            <span class='text' style='width:auto;'>{{domain}}</span>
                                            <span style='font-size:16px;margin-top:2px;' class='form-bt-remove el-icon el-icon-operation-delete' @click.prevent='lgwRemoveBind(index)'></span>
                                        </div>
                                    </el-form-item>
                                </div>
                                <el-form-item prop="lgwImsiRange" v-show=false>
                                    <el-input v-model="networkForm.lgwImsiRange"></el-input>
                                </el-form-item>
                            </template>
                        </template>
                    </div>
                </el-collapse-item>	
            </el-collapse>
        </el-form>
    </div>
	<div class='addSlide' id='ipsecAddPanel' v-show="showIpsecAdd"></div>

	<!-- wan config 修改 弹窗 -->
    <el-dialog class="gnbConfigAddDialog" top="25vh" title="Modify WAN Config" width="50%" :visible.sync="addWanConfigDialogShow" @close="closeAddWanConfigDialog" :close-on-click-modal="false" :modal-append-to-body="false" >
        <el-form ref="addWanConfigForm" :model='addWanConfigForm' :rules='addWanConfigRules' label-position="top">
            <el-form-item prop='index' label="Index" label-width="160px" style='width: 45%;'>
                <el-input v-model.trim='addWanConfigForm.index' disabled></el-input>
            </el-form-item>
            <el-form-item prop='wanName' label="WAN Name" label-width="160px" style='width: 45%;'>
                <el-input v-model.trim='addWanConfigForm.wanName' disabled> </el-input>
            </el-form-item>
            <el-form-item v-if='addWanConfigForm.index != "1"' prop='effectEnable' label='<%=rb.getString("ShiFouShengXiao")%>' label-width="160px" style='width: 45%;'>
                <el-switch v-model="addWanConfigForm.effectEnable" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#BDC1C6"></el-switch>
            </el-form-item>
            <el-form-item prop='ipAccessMode' label="IP Access Mode" label-width="160px" style='width: 45%;'>
                <el-select v-model='addWanConfigForm.ipAccessMode' style="width:60px;padding-top:5px;">
                    <el-option label='DHCP' value='0'></el-option>
                    <el-option label='Static IP' value='1'></el-option>
                    <el-option label='IPV6 DHCP' value='3' v-if="!['Intel_CR_CA','Intel_CR_SC','Intel_CR_TC','Intel_CR_DC','Intel_CR','MLN_SC','MLN_CA','MLN_DC','MLN'].includes(platformType)"></el-option>
                    <el-option label='IPV6 Static IP' value='4' v-if="!['Intel_CR_CA','Intel_CR_SC','Intel_CR_TC','Intel_CR_DC','Intel_CR','MLN_SC','MLN_CA','MLN_DC','MLN'].includes(platformType)"></el-option>
                </el-select>
            </el-form-item>
            <el-form-item v-if='addWanConfigForm.ipAccessMode == "1"' prop='ipAddress' label="IP Address" label-width="160px" style='width: 45%;'>
                <el-input v-model.trim='addWanConfigForm.ipAddress'> </el-input>
            </el-form-item>
            <el-form-item v-if="addWanConfigForm.ipAccessMode == '4' && !['Intel_CR_CA','Intel_CR_SC','Intel_CR_TC','Intel_CR_DC','Intel_CR','MLN_SC','MLN_CA','MLN_DC','MLN'].includes(platformType)" prop='ipv6IpAddress' label="IP Address" label-width="160px" style='width: 45%;'>
                <el-input v-model.trim='addWanConfigForm.ipv6IpAddress'> </el-input>
            </el-form-item>
            <el-form-item v-if='addWanConfigForm.ipAccessMode == "1"' prop='netmask' label="Netmask" label-width="160px" style='width: 45%;'>
                <el-input v-model.trim='addWanConfigForm.netmask'> </el-input>
            </el-form-item>
            <el-form-item v-if='addWanConfigForm.ipAccessMode == "1"' prop='gateway' label="Gateway" label-width="160px" style='width: 45%;'>
                <el-input v-model.trim='addWanConfigForm.gateway'> </el-input>
            </el-form-item>
            <el-form-item v-if="addWanConfigForm.ipAccessMode == '4' && !['Intel_CR_CA','Intel_CR_SC','Intel_CR_TC','Intel_CR_DC','Intel_CR','MLN_SC','MLN_CA','MLN_DC','MLN'].includes(platformType)" prop='prefix' label="Prefix" label-width="160px" style='width: 45%;'>
                <el-input v-model.trim='addWanConfigForm.prefix'> </el-input>
            </el-form-item>
            <el-form-item v-if="addWanConfigForm.ipAccessMode == '4' && !['Intel_CR_CA','Intel_CR_SC','Intel_CR_TC','Intel_CR_DC','Intel_CR','MLN_SC','MLN_CA','MLN_DC','MLN'].includes(platformType)" prop='ipv6Gateway' label="IPV6 Gateway" label-width="160px" style='width: 45%;'>
                <el-input v-model.trim='addWanConfigForm.ipv6Gateway'> </el-input>
            </el-form-item>

            <el-form-item v-if='addWanConfigForm.ipAccessMode == "0"' prop='option60' label="Option60" label-width="160px" style='width: 45%;' class='validate-item'>
                <el-input v-model.trim='addWanConfigForm.option60'>
                    <template slot="append">Range:0~64 Digit</template>
                </el-input>
            </el-form-item>
            <el-form-item prop='vlanId' label="VLAN ID" label-width="160px" style='width: 45%;' class='validate-item'>
                <el-input v-model.trim='addWanConfigForm.vlanId'>
                    <template slot="append">Range:1~4094 Integer</template>
                </el-input>
            </el-form-item>
        </el-form>
        <div slot="footer" class="importFooter">
            <el-button type="primary" @click="addWanConfigDialogSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="addWanConfigDialogShow = false"><%=rb.getString("QuXiao")%></el-button>
         </div>
    </el-dialog>

    <el-dialog class="gnbConfigAddDialog" top="25vh" width="50%" 
        title="Modify Route" 
        :visible.sync="routeDlShow" 
        :close-on-click-modal="false" 
        :modal-append-to-body="false" 
        @close="closeRouteDl">
        <el-form ref="routeModify" :model='routeForm' :rules='routeRules' label-position="top">
            <el-form-item v-show="false" prop='index' label="Index" label-width="160px" style='width: 45%;'>
                <el-input v-model.trim='routeForm.index' disabled></el-input>
            </el-form-item>
            <el-form-item prop='routeEffective' label="Effective or Not" label-width="160px" style='width: 45%;'>
                <el-switch v-model="routeForm.routeEffective" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#BDC1C6"></el-switch>
            </el-form-item>
            <el-form-item prop='routeNetwork' label="Network" label-width="160px" style='width: 45%;'>
                <el-input v-model.trim='routeForm.routeNetwork'></el-input>
            </el-form-item>
            <el-form-item prop='routeNetmask' label="Netmask" label-width="160px" style='width: 45%;'>
                <el-input v-model.trim='routeForm.routeNetmask'></el-input>
            </el-form-item>
            <el-form-item prop='routeGateway' label="Gateway" label-width="160px" style='width: 45%;'>
                <el-input v-model.trim='routeForm.routeGateway'></el-input>
            </el-form-item>
        </el-form>
        
        <div slot="footer" class="importFooter">
            <el-button type="primary" @click="routeDlSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="routeDlShow = false"><%=rb.getString("QuXiao")%></el-button>
        </div>
    </el-dialog>
</div>
<script>
	var networkVm = new Vue({
		el:'#networkPanel',
		data(){
		    var vm = this, regIPAddress = /^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-4]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/;
			var validateRange = (rule,value,callback)=>{
				var min = rule.min;
				var max = rule.max;
				var reg = /^(\d+\.\.){0,1}(\d+)$/;

                if(['Intel_CR_CA','Intel_CR_SC','Intel_CR_TC','Intel_CR_DC','Intel_CR','MLN_SC','MLN_CA','MLN_DC','MLN'].includes(vm.platformType)){
                    callback();
                }else{ 
                    if(value == '' || value == undefined || value == null){
                        callback(new Error('<%=rb.getString("LengthGeShiCuoWu")%>'))
                    }else{
                        if(!reg.test(value) || value < min || value > max){
                            callback(new Error('<%=rb.getString("LengthGeShiCuoWu")%>'))
                        }else{
                            callback();
                        }
                    }
                }
			},
			validateDnsAddress = (rule,value,callback)=>{
                if(value == '' || value == undefined || value == null){
                    callback();
                }else{
                    if(!regIPAddress.test(value)){
                        callback(new Error('<%=rb.getString("LengthGeShiCuoWu")%>'))
                    }else{
                        if(this.networkForm.dnsAddress == this.networkForm.dnsAddress2){
                            callback(new Error('<%=rb.getString("YuMingDiZhiChongFu")%>'))
                        }else{
                            callback();
                        }
                    }
                }
            },
            validateDnsAddress2 = (rule,value,callback)=>{
                if(value == '' || value == undefined || value == null){
                    callback();
                }else{
                    if(!regIPAddress.test(value)){
                        callback(new Error('<%=rb.getString("LengthGeShiCuoWu")%>'))
                    }else{
                        this.$refs.networkForm.validateField('dnsAddress');
                        callback();
                    }
                }
            },
			validateSubnetMask = (rule,value,callback)=>{
                if(['Intel_CR_CA','Intel_CR_SC','Intel_CR_TC','Intel_CR_DC','Intel_CR','MLN_SC','MLN_CA','MLN_DC','MLN'].includes(vm.platformType)){
                    callback();
                }else{ 
                    if(value == '' || value == undefined || value == null){
                        callback(new Error('<%=rb.getString("LengthGeShiCuoWu")%>'))
                    }else{
                        if(!regIPAddress.test(value)){
                            callback(new Error('<%=rb.getString("LengthGeShiCuoWu")%>'))
                        }else{
                            callback();
                        }
                    }
                }
            },
            validRouteIP = (rule,value,callback)=>{
                if(value == '' || value == undefined || value == null){
                    if(rule.required == true) {
                        callback(new Error('<%=rb.getString("LengthGeShiCuoWu")%>'))
                    }else {
                        callback();
                    }
                }else{
                    if(!regIPAddress.test(value)){
                        callback(new Error('<%=rb.getString("LengthGeShiCuoWu")%>'))
                    }else{
                        var list = vm.routerList,
                            index = vm.routeForm.index,
                            routeNetwork = vm.routeForm.routeNetwork,
                            other = list.filter(function(item, idx){
                                return (item.index||idx) != index && item.routeNetwork == routeNetwork;
                            });
                        
                        if(rule.validRepeat && other.length) {
                            callback('<%=rb.getString("YiCunZai")%>');
                        }else {
                            callback();
                        }
                    }
                }
            },
            validRouteMask = (rule,value,callback)=>{
                if(value == '' || value == undefined || value == null){
                    if(rule.required == true) {
                        callback(new Error('<%=rb.getString("LengthGeShiCuoWu")%>'))
                    }else {
                        callback();
                    }
                }else{
                    if(!isIPv4(value) && value != '255.255.255.255'){
                        callback(new Error('<%=rb.getString("LengthGeShiCuoWu")%>'))
                    }else{
                        var list = vm.routerList,
                            index = vm.routeForm.index,
                            routeNetwork = vm.routeForm.routeNetwork,
                            other = list.filter(function(item){
                                return item.index != index && item.routeNetwork == routeNetwork;
                            });
                        
                        if(rule.validRepeat && other.length) {
                            callback('<%=rb.getString("YiCunZai")%>');
                        }else {
                            callback();
                        }
                    }
                }
            },
            validateOption60 = (rule,value,callback)=>{
                var min = rule.min;
                var max = rule.max;
                var reg = /^(\d+\.\.){0,1}(\d+)$/;
                if(value == '' || value == undefined || value == null){
                    callback();
                }else{
                    if(!reg.test(value) || value < min || value > max){
                         callback(new Error('<%=rb.getString("LengthGeShiCuoWu")%>'))
                    }else{
                        callback();
                    }
                }
            },
            validateVlanId = (rule,value,callback)=>{
                var min = rule.min;
                var max = rule.max;
                var reg = /^(\d+\.\.){0,1}(\d+)$/;
                if(value == '' || value == undefined || value == null){
                    callback();
                }else{
                    if(!reg.test(value) || value < min || value > max){
                         callback(new Error('<%=rb.getString("LengthGeShiCuoWu")%>'))
                    }else{
                        callback();
                    }
                }
            },
            validateNetmask = (rule,value,callback)=>{
                if(['Intel_CR_CA','Intel_CR_SC','Intel_CR_TC','Intel_CR_DC','Intel_CR','MLN_SC','MLN_CA','MLN_DC','MLN'].includes(vm.platformType)){
                    callback();
                }else{ 
                    if(vm.addWanConfigForm.ipAccessMode == '1'){
                        if(vm.isMask(value)){
                            callback();
                        }else{
                            callback(new Error('<%=rb.getString("QingShuRuHeFaDeYanMa")%>'))
                        }
                    }else if(vm.addWanConfigForm.ipAccessMode == '4'){
                        if(vm.isNumeric(value)&&parseInt(value)>=0 && parseInt(value)<=128){
                            callback();
                        }else{
                            callback(new Error('<%=rb.getString("FanWei")%>：0~128,Integer'))
                        }
                    }else{
                        callback();
                    }
                }
            },
            validateGateway = (rule,value,callback) => {
                if(['Intel_CR_CA','Intel_CR_SC','Intel_CR_TC','Intel_CR_DC','Intel_CR','MLN_SC','MLN_CA','MLN_DC','MLN'].includes(vm.platformType)){
                    callback();
                }else{                
                    if(vm.addWanConfigForm.ipAccessMode == '1'){
                    if(value == '' || value == undefined || value == null){
                            callback();
                    }else{
                        if(vm.isValidIP(value)){
                            callback();
                        }else{
                            callback(new Error('<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>'))
                        }
                    }
                    }else if(vm.addWanConfigForm.ipAccessMode == '4'){
                        if(vm.isIPv6(value)){
                            callback();
                        }else{
                            callback(new Error('<%=rb.getString("QingShuRuHeFaDeIPV6DiZhi")%>'))
                        }
                    }else{
                        callback();
                    }
                }
            },
            validateIPAddress = (rule,value,callback)=>{
                 if(value == '' || value == undefined || value == null){
                     callback(new Error('<%=rb.getString("LengthGeShiCuoWu")%>'))
                 }else{
                    // wanConfigTableList
                    var ipList = vm.wanConfigTableList.map(function(item){
                            if(item.index != vm.addWanConfigForm.index){
                                return item.ipAddress;
                            }
                        });
                    if(!regIPAddress.test(value)){
                        callback(new Error('<%=rb.getString("LengthGeShiCuoWu")%>'))
                    }else if(ipList.includes(value)) {
                        callback(new Error('<%=rb.getString("IPYiCunZai")%>'));
                    }else{
                        callback();
                    }
                 }
            },
            validateIpv6IpAddress = (rule,value,callback)=>{
                if(['Intel_CR_CA','Intel_CR_SC','Intel_CR_TC','Intel_CR_DC','Intel_CR','MLN_SC','MLN_CA','MLN_DC','MLN'].includes(vm.platformType)){
                    callback();
                }else{
                    if(value == '' || value == undefined || value == null){
                        callback(new Error('<%=rb.getString("LengthGeShiCuoWu")%>'))
                    }else{
                        // wanConfigTableList
                        var ipList = vm.wanConfigTableList.map(function(item){
                                if(item.index != vm.addWanConfigForm.index){
                                    return item.ipv6IpAddress;
                                }
                            });
                        
                        if(!vm.isIPv6(value)){
                            callback(new Error('<%=rb.getString("LengthGeShiCuoWu")%>'))
                        }else if(ipList.includes(value)) {
                            callback(new Error('<%=rb.getString("IPYiCunZai")%>'));
                        }else{
                            callback();
                        }
                    }
                }
            },
            validateLgwIp = (rule,value,callback)=>{
                var min = rule.min;
                var max = rule.max;
                var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/,
                    serverReg = /^[a-zA-Z0-9-_]+(\.[a-zA-Z0-9-_]+)+$/;
                if(value == '' || isValidIP(value) || serverReg.test(value)) {
                    if(rule.code == 'lgw') {
                        vm.lgwIsAllBindIpInRange();
                    
                        var pool = vm.networkForm.lgwIpPool,
                            mask = vm.networkForm.lgwIpPoolNetmask,
                            firstIp = vm.networkForm.lgwFirstIp,
                            lastIp = vm.networkForm.lgwLastIp;
                        if(isValidIP(value)) {
                            if(pool && isValidIP(pool) && mask) {
                                var startIp = lgwGetLowAddr(pool, mask),
                                    endIp = lgwGetHighAddr(pool, mask),
                                    ipRangeTip = '<%=rb.getString("IPBangDingFanWei")%>' + startIp + '-' + endIp;    
                                if(vm.lgwCompareIp(value,startIp,endIp)) {
                                    if(lastIp && isValidIP(value)) {
                                        var startNum = vm.lgwChangeIpToNum(firstIp),
                                            endNum = vm.lgwChangeIpToNum(lastIp);
                                        
                                        if(startNum - endNum >= 0) {
                                            callback('First IP should less than Last IP')
                                        }else{
                                            callback();
                                        }
                                    }else {
                                        callback();
                                    }
                                }else {
                                    callback(ipRangeTip)
                                }

                            }else {
                                callback();
                            }
                        }else {
                            callback('Please input valid IP Address')
                        }
                    }else{
                        callback();
                    }
                }else{
                    callback(new Error('Please input valid IP Address'))
                }
            }
            validateLgwImsiRange = (rule,value,callback)=>{
                var bool = vm.lgwIsAllBindIpInRange();

                if(bool) {
                    callback();
                }else {
                    callback('IMSI to IP Binding exist out range of Static IP')
                }
            };
			return{
				isBaiblx_QRTB: settingVue.selectedRow.platformType == 'BLX' && settingVue.selectedRow.network_model == 'TDDMode',
				isBaiblx_BLQ: settingVue.selectedRow.platformType == 'BLX' && settingVue.selectedRow.network_model == 'FDDMode',
				isMLQ: settingVue.selectedRow.product == 'MLQ',

				rebootMap: {},

				delRecord: {
					ipsecList: [],
				},

				codeTableList: [],
				oldTbList: [],
				wanConfigTableList: [],
                slaveInterfaceList: [],
				networkForm:{
				    connectType: '',
                    //4860
                    linkSpeedNegotiated: '',
                    lgwEnable:'',
                    lgwMode:'',
                    lgwIpPool:'',
                    lgwIpPoolNetmask:'',
                    lgwIpEnable:'',
                    lgwFirstIp:'',
                    lgwLastIp:'',
                    lgwImsiRange:'',

                    ipAccessMode1: '',
                    ipAddress1: '',
                    netmask1: '',
                    prefix1: '', //prefix
                    gateway1: '',
                    ipv6IpAddress1:'',
                    ipv6Gateway1: '',//ipv6Gateway
                    vlanId1: '',
                    //effectEnable1: '',
                    option601: '',

                    ipAccessMode2: '',
                    ipAddress2: '',
                    netmask2: '',
                    gateway2: '',
                    prefix2: '',
                    ipv6IpAddress2: '',
                    ipv6Gateway2: '',
                    vlanId2: '',
                    effectEnable2: '',
                    option602: '',

                    ipAccessMode3: '',
                    ipAddress3: '',
                    netmask3: '',
                    gateway3: '',
                    prefix3: '',
                    ipv6IpAddress3: '',
                    ipv6Gateway3: '',
                    vlanId3: '',
                    effectEnable3: '',
                    option603: '',

                    ipAccessMode4: '',
                    ipAddress4: '',
                    netmask4: '',
                    gateway4: '',
                    prefix4: '',
                    ipv6IpAddress4: '',
                    ipv6Gateway4: '',
                    vlanId4: '',
                    effectEnable4: '',
                    option604: '',

                    ipAccessMode5: '',
                    ipAddress5: '',
                    netmask5: '',
                    gateway5: '',
                    prefix5: '',
                    ipv6IpAddress5: '',
                    ipv6Gateway5: '',
                    vlanId5: '',
                    effectEnable5: '',
                    option605: '',

                    ipAccessMode6: '',
                    ipAddress6: '',
                    netmask6: '',
                    gateway6: '',
                    prefix6: '',
                    ipv6IpAddress6: '',
                    ipv6Gateway6: '',
                    vlanId6: '',
                    effectEnable6: '',
                    option606: '',

                    ipAccessMode7: '',
                    ipAddress7: '',
                    netmask7: '',
                    gateway7: '',
                    prefix7: '',
                    ipv6IpAddress7: '',
                    ipv6Gateway7: '',
                    vlanId7: '',
                    effectEnable7: '',
                    option607: '',

                    ipAccessMode8: '',
                    ipAddress8: '',
                    netmask8: '',
                    gateway8: '',
                    prefix8: '',
                    ipv6IpAddress8: '',
                    ipv6Gateway8: '',
                    vlanId8: '',
                    effectEnable8: '',
                    option608: '',

                    ipAccessMode9: '',
                    ipAddress9: '',
                    netmask9: '',
                    gateway9: '',
                    prefix9: '',
                    ipv6IpAddress9: '',
                    ipv6Gateway9: '',
                    vlanId9: '',
                    effectEnable9: '',
                    option609: '',

                    ipAccessMode10: '',
                    ipAddress10: '',
                    netmask10: '',
                    gateway10: '',
                    prefix10: '',
                    ipv6IpAddress10: '',
                    ipv6Gateway10: '',
                    vlanId10: '',
                    effectEnable10: '',
                    option6010: '',

                    ipAccessMode11: '',
                    ipAddress11: '',
                    netmask11: '',
                    gateway11: '',
                    prefix11: '',
                    ipv6IpAddress11: '',
                    ipv6Gateway11: '',
                    vlanId11: '',
                    effectEnable11: '',
                    option6011: '',

                    ipAccessMode12: '',
                    ipAddress12: '',
                    netmask12: '',
                    gateway12: '',
                    prefix12: '',
                    ipv6IpAddress12: '',
                    ipv6Gateway12: '',
                    vlanId12: '',
                    effectEnable12: '',
                    option6012: '',

                    dnsConfigEnable: '',
                    dnsAddress: '',
                    dnsAddress2: '',

					mtu:'',
                    accessLMTViaWAN: '',
                    quickInterfaceBinding: '',
                    slaveInterface: '',
                    subMachineAddress: '',

					lanIpAddress: '',
					subnetMask: '',

					ipsecEnable:'',
                    ipsecList:[],

                    // Static Routing
                    // route 第一组
                    routeEffective1: '',
                    routeNetwork1: '',
                    routeNetmask1: '',
                    routeGateway1: '',

                    // route 第二组
                    routeEffective2: '',
                    routeNetwork2: '',
                    routeNetmask2: '',
                    routeGateway2: '',
                    
                    // route 第三组
                    routeEffective3: '',
                    routeNetwork3: '',
                    routeNetmask3: '',
                    routeGateway3: '',
                    
                    // route 第四组
                    routeEffective4: '',
                    routeNetwork4: '',
                    routeNetmask4: '',
                    routeGateway4: '',
                    
                    // route 第五组
                    routeEffective5: '',
                    routeNetwork5: '',
                    routeNetmask5: '',
                    routeGateway5: '',
                    
                    // route 第六组
                    routeEffective6: '',
                    routeNetwork6: '',
                    routeNetmask6: '',
                    routeGateway6: '',
                    
                    // route 第七组
                    routeEffective7: '',
                    routeNetwork7: '',
                    routeNetmask7: '',
                    routeGateway7: '',
                    
                    // route 第八组
                    routeEffective8: '',
                    routeNetwork8: '',
                    routeNetmask8: '',
                    routeGateway8: '',
                    
                    // route 第九组
                    routeEffective9: '',
                    routeNetwork9: '',
                    routeNetmask9: '',
                    routeGateway9: '',
                    
                    // route 第十组
                    routeEffective10: '',
                    routeNetwork10: '',
                    routeNetmask10: '',
                    routeGateway10: '',
                    
                    // route 第十一组
                    routeEffective11: '',
                    routeNetwork11: '',
                    routeNetmask11: '',
                    routeGateway11: '',
                    
                    // route 第十二组
                    routeEffective12: '',
                    routeNetwork12: '',
                    routeNetmask12: '',
                    routeGateway12: '',
				},
				networkRules:{
					mtu:[{required:true, validator:validateRange, min:700,max:1600}],
					dnsAddress:[{validator: validateDnsAddress}],
					dnsAddress2:[{validator: validateDnsAddress2}],
					lanIpAddress:[{required: true, validator: validateSubnetMask}],
					subnetMask:[{required: true, validator: validateSubnetMask}],

                    subMachineAddress: [{validator: validateDnsAddress}],

                    lgwIpPool:[{validator: validateLgwIp}],
                    //lgwIpPoolNetmask:[{validator: validateLgwIp}],
                    lgwFirstIp:[{validator: validateLgwIp, code: 'lgw'}],
                    lgwLastIp:[{validator: validateLgwIp, code: 'lgw'}],
                    lgwImsiRange: [{validator: validateLgwImsiRange}]
				},
				activeNames:['wan','staticRouter','ipsec','lgw'],
				ipsecData:[],
				showIpsecAdd:false,
                
                codeList:[],
				operType:'',
				rowData:[],
				smallCellCode:'',
				addIpsecFlag:true,
                
                quickInterfaceBindingList:[],
                newQuickBandList: [],
				addWanConfigDialogShow:false,
                addWanConfigForm:{
                    index:'',
                    wanName:'',
                    ipAccessMode:'0',
                    ipAddress:'',
                    
                    netmask:'',
                    gateway:'',
                    
                    ipv6IpAddress: '',
                    prefix: '',
                    ipv6Gateway:'',
                    vlanId:'',
                    effectEnable: '',                    
                    option60:'',
                },
                addWanConfigRules:{
					ipAddress:[{required: true, validator: validateIPAddress}],
					netmask:[{required: true, validator: validateNetmask}],
					gateway:[{validator: validateGateway}],
					prefix:[{required: true, validator: validateNetmask,min:0,max:128}],
                    ipv6Gateway:[{required: true, validator: validateGateway}],
                    ipv6IpAddress:[{required: true, validator: validateIpv6IpAddress}],
					vlanId:[{validator:validateVlanId, min:1,max:4094}],
                    option60:[{validator:validateOption60, min:0,max:64}],
                },
                platformType: '',
                wanConfigNumber: '',

                routerList: [],
                routeForm: {
                    index: '',
                    routeEffective: '',
                    routeNetwork: '',
                    routeNetmask: '',
                    routeGateway: '',
                },
                routeRules: {
                    routeNetwork:[{validator: validRouteIP, validRepeat: true}],
					routeNetmask:[{required: true, validator: validRouteMask}],
					routeGateway:[{validator: validRouteIP}],
                },
                routeDlShow: false,

                lgwBindCls:'',
                lgwBindGroup:[],
				lgwBindImsi:'',
				lgwBindIp:'',
				casts:{
					'78D5558D597F0E279703CDD534C3F4ED':'mtu',
					'0DD9F32B4EE01BD187C6A2291235AAF6':'ipsecEnable',
					'0D3FD79C1A7FACBDA596A3F7E2B269DA':'ipsecList',
					'D2334B15F9526C68FF442B347B646998':'index',
					'E55003AB9D0569A13DDE8CD1A79C184C':'configEnable',
					'625F6AE6F16D82811BEBF21AA117084A':'leftAuth',
					'8A8F71D4D40F3ECC0E157AEEF52C66FC':'rightAuth',
					'D72FBBE5FFCE11F54C9CAC0F9097ECD8':'gateway',
					'CFB89281F2F56558BD14DD5667939C95':'rightSubnet',
					'7D739CEE484A613759290C643FE32794':'leftId',
					'8676CCE229CF917FDF83A173A9B3D157':'rightId',
					'4A4FED52283D2485889266A91BBF7FB9':'leftCert',
					'6A2B9AC3C3C4EC8DC4DEBD8F11735081':'secretKey',
					'7DC96CEB2C84C2CECD06FA2BEC88101F':'leftSourceIp',
					'3D6E50F57A24196E9E9283DB51F25A3D':'leftSubnet',
					'7749E8211E895EAAD0342EC858203E02':'ikeEncryption',
					'C819AB06DF0A261A8AE6889684A42007':'ikeDhGroup',
					'988D286A5E933C112A5DC0C73EB989C5':'ikeAuthentication',
					'16CF381CD2E3C63468E193B4F7EF86AD':'espEncryption',
					'70CD10EF601A64E5A416B33A3044DEED':'espDhGroup',
					'C94EC65F3066B2B16A949245C4D8F262':'espAuthentication',
					'3307DACE172E5D849721007EE1850983':'fragmentation',
					'585AED8C45BAB306E32E10A137E8CA85':'keylife',
					'EE6FF7BD0D01DF5F34F000B4E5FE84B1':'IKELifeTime',
					'11E5E064D57576132BCD63CEE21B13BD':'RekeyMargin',
					'079676AFCEEA160FF937C14E0D165DE1':'Dpdaction',
					'3D8C82590ED4B3211835558B4CFA0BF7':'Dpddelay',
					
					'E6061BBDD52E48377DDFA383A6E0C004':'mtu',
					'76B3B3F21B6DE3E94B5CD9D02EDAE652':'ipsecEnable',
					'639F06803FCB53E02F4BF83636416C8A':'ipsecList',
					'287F9B9D8A75A7FB6C1A305ECFFAFA9C':'index',
					'EC9142ADED4F7087FE70188BFCC22ED8':'configEnable',
					'7B840CF742517CE3EF69ACE45B466BC4':'leftAuth',
					'746693F8FB895B742CABFFEB31D99096':'rightAuth',
					'89C46078159A8688E3D906A5968DACCA':'gateway',
					'A3551DF61BA9689ABE846793FAD0113F':'rightSubnet',
					'FC5B6690BAE93A8A135339EAD9B9E546':'leftId',
					'FA42B8FE844FC4C85AC3CBFFD5E93B8E':'rightId',
					'53D6BE979061A4C28816E0DFE83ADFF5':'leftCert',
					'235F1044BACD47159BF580787A60D18E':'secretKey',
					'782613C63509F882310D25E7D6BAD258':'leftSourceIp',
					'6F03C772C49DAA1EFF7A064EC1DBE488':'leftSubnet',
					'26C103BE8685A76D2F3B7B90BB386118':'ikeEncryption',
					'96FBA47C6EBC4092665E65BFF5119396':'ikeDhGroup',
					'61F68EDA92EEEF68617ACAE3FAF7168A':'ikeAuthentication',
					'A8B64CD85E3120C44F405BD6A2348F28':'espEncryption',
					'E017AE44AAC015840D569838D97B408A':'espDhGroup',
					'3BBAE68BDEA350642896EFC17E887995':'espAuthentication',
					'F93EB368E466EF0A5FEECFAD8117826E':'fragmentation',
					'E26C7B2A93D842AE240D4006A1B82839':'keylife',
					'26FD4B8099268969FB6B1FA0304C4E62':'IKELifeTime',
					'9C5B3E14F0E5D96082EBAA2BE2A090CA':'RekeyMargin',
					'B0EE2D98F2A295F0B25D0CA6197B5D1B':'Dpdaction',
					'E305C675A62C6DD00137AF9735294EEA':'Dpddelay',

					'5F1C82D000BCB635228F17A63879C3AF':'mtu',
					'AA8138ECF5F9D7E8947775EED98819C5':'ipsecEnable',
					'E349C0000B87AF949A94C0F3E34D229B':'ipsecList',
					'BF1830618187016FBBDFDB8A9CDBDCCF':'index',
					'66BA222E3C36F9C84F6B96F93A8B65F9':'configEnable',
					'D8C6E552A9E115D1640735A2E824D168':'leftAuth',
					'55981661C2689462E533BFD120728E57':'rightAuth',
					'4A58FB0B478EC42495675106EE3FED16':'gateway',
					'BCB818C983DF6A6B0EE217001B72F80D':'rightSubnet',
					'4963796C1324E060D66BA18644014052':'leftId',
					'DD51162517201DC84CBFC63B56F13469':'rightId',
					'50E35400B1DB12824FA59718B8FC88D6':'leftCert',
					'356509B7C038629A70797CA6A09ADADA':'secretKey',
					'1BC08B9994E0079A75CBC679BAE2ED8D':'leftSourceIp',
					'49E0B1A93272302EC09963229875DCC6':'leftSubnet',
					'4DFFF72589EC4484E046C5240FE73D61':'ikeEncryption',
					'FDCB7C73BA08A17169EA3570D1DBCA82':'ikeDhGroup',
					'755AE492661CEB5BC3D315214FC8AC4D':'ikeAuthentication',
					'5BA3ACFD8C6368941CE95A998622E6FF':'espEncryption',
					'46D1433A94533707BB485D28A23A792F':'espDhGroup',
					'E5521F4BB2DC068734129D128AAD8502':'espAuthentication',
					'65575D0974B8AF591DB20DF3481D70D5':'fragmentation',
					'EA4597564ED35A3B546881B20F51FC31':'keylife',
					'A52379A3A5BE1E0A3C2CC49D5490FFC5':'IKELifeTime',
					'63907269121A2CB8A2462D7CB361B769':'RekeyMargin',
					'CAB1A8155102125B2D8B491384694CA8':'Dpdaction',
					'D79F7C3004B6A7E077E017D35BFE4CAA':'Dpddelay',

                    //4860模式 ipsec setting
                    //'':'mtu', // 此模式是否含有该字段
                    'F62DAA47BF17896CF11905B17C2224E4':'lgwEnable',
                    '787888F6AFEF00BCAF8AD3CD28CB5656':'lgwMode',
                    'AC0E16101E86CD6C14EA02FBDDFD9E1A':'lgwIpPool',
                    '618FF692228D6424006A875639DCA655':'lgwIpPoolNetmask',
                    '20847D52FC3E1D6EBB14FBD49D0C2121':'lgwIpEnable',
                    'F2DD8A56743EE443626948366CEE214A':'lgwFirstIp',
                    'AD092825F8E817757CABA783965CDD14':'lgwLastIp',
                    '9F497D0576E74E94EB6C68F4E7C7CADA':'lgwImsiRange',

                    '9B44673D0845D333D4C22B5E31CA2AD5':'ipsecEnable',
					'54C26FB9A9A8E590156CA1AE142347E6':'ipsecList',
					'912536A44A4157830D7ED10FC73231F1':'index',
					'3EFC352669502E1215C2FE5AFA5A6ACE':'configEnable',
					'DE575284C103AF1CE811CDE53D3671D2':'leftAuth',
					'16C363A1F91435EEC6865728431C9CC2':'rightAuth',
					'3F0C8C384EBC78C5DDBFA9A4F042FBF3':'gateway',
					'2AF0E322EED376751C2A89C70A3EAA58':'rightSubnet',
					'F72424AD50E0ED3177A54FD9D3C6F4A3':'leftId',
					'7476985E29C4D7063A7B10AFC05AFCA0':'rightId',
					'43779DE197608732C641BAFCD334D7FF':'leftCert',
					'C6FCBE45C4C9BFFF3AB5E427A2ECCE87':'secretKey',
					'ADF0FB90829797227BB58FBDAB1C1CD3':'leftSourceIp',
					'DE043C9775F5F77A2A331DE060E49163':'leftSubnet', 
					'F57757CF1A0DB57C09F16D2F0BF1B1C0':'ikeEncryption',
					'5D2D5F16D81C4997A0D52EC53A962914':'ikeDhGroup',
					'06CF8E6184A22F1FA1D57F0490B50F1D':'ikeAuthentication',
					'541B545CF8E6053018FD30FE1E4C95B6':'espEncryption',
					'3D71307AF6C589890D06538A09A7F6FA':'espDhGroup',
					'F08A3DBB64EBE52389AA4E2503AAE945':'espAuthentication',
					'CC7CFB4A508C5C8F0A03421B7DCFEF1B':'fragmentation',
					'96810BF855B560121963CD7FFF9437AE':'keylife',
					'B97DB4B8A281C1B4515F1B083C97992D':'IKELifeTime',
					'64D87408627039CD1367BBB31684726D':'RekeyMargin',
					'C94A7ACD5822980C97344497A9F8C310':'Dpdaction',
					'BB61F27ACC29DAD8A18CD3662F835A2C':'Dpddelay',

                     //MLN模式 ipsec setting
                     //'':'mtu', // 此模式是否含有该字段
                    '30816C18CC1E1BFEC1F8D205AB29A44C':'lgwEnable',
                    '7433A424A75F34DFDCF1C69E85B2D139':'lgwMode',
                    '370612861F5268CB05ABEE0A8090B706':'lgwIpPool',
                    '9F00F922FA61DC5C43CD29AE6DF82F11':'lgwIpPoolNetmask',
                    '03C55D64392ED9EAD5089C29D7B8108D':'lgwIpEnable',
                    '5E90CF67977AE4183E52A2357002D33D':'lgwFirstIp',
                    'C7EE712C78FE10E5309DD7783D4990D7':'lgwLastIp',
                    'CE0B17E02815826DE3CD9CAE836208F7':'lgwImsiRange',

                    '95BE1BEDF6078C3E445A66EBDAEFAAA8':'ipsecEnable',
                    '57A1B3CAAA3257A2E15668400FD32E56':'ipsecList',
                	'FB4564F25F0CA9641FDFC9B7B6C57D1C':'index',
                	'90FF159786F7F61658B348646E465E04':'configEnable',
                	'2206063744D53ABE6A4C816DBE908AD5':'leftAuth',
                	'0FA9D8CE4C716A21824DAE46440EC6D1':'rightAuth',
                	'E5BF3EC39B27940FF051B93C92129466':'gateway',
                	'0523EDACDE00A88D513480A9B7F8B014':'rightSubnet',
                	'71ACF0410C3192339BDDF8161B5A25A9':'leftId',
                	'0CA26E7E4A90C81C532AC926B2B6E427':'rightId',
                	'C0709D92868AC9B241920A9BA16BCD7C':'leftCert',
                	'6BB85E45E90A45AE58C668EFD481AACC':'secretKey',
                	'E54E67B67A718D10772D3D71F8ACCF07':'leftSourceIp',
                	'AE713FA1346F2BF1CBA4C56AFF08FB4B':'leftSubnet',
                	'8F0E25DC3F7D40EAB76E446B9D26AB28':'ikeEncryption',
                	'F849F8772354740B1CFB81E1786C4B6C':'ikeDhGroup',
                	'29B3F5CDA2D9B3CFAEFE40F9E76000AF':'ikeAuthentication',
                	'225178C844869309C1A0F5A33836114A':'espEncryption',
                	'8639CB8454A47CF27D4E8B9ED0F06623':'espDhGroup',
                	'6CAE67E19008FF422C91B4CEFECB4FD4':'espAuthentication',
                	'4E714D20EC95CEEB80E5D65C97D1FBCA':'fragmentation',
                	'E1C6C559FCD69807F92C880B7A6BBCC6':'keylife',
                	'DE74626F38119961B69E6A513D873C33':'IKELifeTime',
                	'8E0CC7F68FA8819E33F9323D53416759':'RekeyMargin',
                	'8F94B7256C0902F20A83E4D576B1A712':'Dpdaction',
                	'9BE34B86DC5D24B3A725380DAAA9C519':'Dpddelay',

                    // wan config 
                    '9ED633734F19457CB136B655E5B54FCC':'connectType',
                    '6CA860F4D76FC429D564F69B8B44F1CD':'dnsConfigEnable',
                    'F0C730C63A034C46BB97AFE5E3A927F9':'dnsAddress',
                    '9D2489FD786DB5CBA1113BC1D6E6BC50':'dnsAddress2',
                    'EA4EBD32D037E5AA5133ED237E06D283':'accessLMTViaWAN',
                    'EFFD3E01920A6388F69F6A60B8406659':'quickInterfaceBinding',
					'154D269A38F7875790EA37BDA9B61B35':'lanIpAddress',
					'C416CE0A9648027FEBCBF2C8464D4645':'subnetMask',
					//wan config 这算一组数据  实际接口中无 index wanName 字段
					'5B9B97E9DA854615A0828F5B7ECB7232':'ipAccessMode1',//第一组
					'AA7870FDB0280E6786A9A56F2F8D7272':'ipAddress1',
					'3ABA7559DB54E6055795873D632E16FE':'netmask1',
					'8D4019A909B65159C434C4DAE7496DCC':'prefix1', //Prefix
					'696C40D3A2ADEFED4A72B11C32EFB62D':'gateway1',
                    'CFB1FC9C55214F383F1DA13D6C8E90A1':'ipv6IpAddress1', //ipv6IpAddress
					'5605A965B33D07B488C5702B14B105A1':'ipv6Gateway1', //ipv6Gateway
					'73519DC68C855FBCE0E1F60931305DDB':'vlanId1',
                    //'F021B5B7D8D2DA005DC30FDCE36A3B23':'effectEnable1',//是否生效开关 第一组数据无开关参数
					'FDCA6E368B96A08BC134DF9BAC34E347':'option601',

					'0BC2DA8D9E59444B206B211BEEE3363C':'ipAccessMode2', //第二组
                    '0F50615704795A6E2A90D5B5908828D6':'ipAddress2',
                    'BF0F5BBDADA71A28D6CC4342AD4F6CA4':'netmask2',
                    'A81CBB6D5A0E05A34163C0E75DCCAEE5':'gateway2',
                    'B7567BCA29046B6708D3D296E725FE08':'prefix2',
                    '6381F72BED397D17A8D5D87C37ECCAA2':'ipv6Gateway2',
                    'F381C889670976F706345B6E9FFF642A':'ipv6IpAddress2',
                    '06230CE2624D5555B3561A1B2D293754':'vlanId2',
                    '5617104FFB9F671E022B1F4CE9B9963F':'effectEnable2',
                    '3E0A74A8E541861E0CCCC7DAA114AC8F':'option602',

                    '69398A497706BFB2AF23D8AD668FCEBD':'ipAccessMode3', //第三组
                    '26F5FA7219C3921489DCF77E9486FF70':'ipAddress3',
                    '6EFAA68691C0CFCE3BF7285AAB5C6AA7':'netmask3',
                    '891CB1C82C9C4B79920DE7831E6192CF':'gateway3',
                    'DB136FA2D56E4ABF8B2A03DE22EB0D4F':'prefix3',
                    'C71D3591CE456ED8D4ECA21A1E733321':'ipv6Gateway3',
                    '7D18C4B72DEA132FAA05F92E30E9A5F0':'ipv6IpAddress3',
                    '740F2E0479325271933E46BEB8B6EAEA':'vlanId3',
                    'BDAA187C67E5BEE472CE02BAEC3E146F':'effectEnable3',
                    '26C9CD07717C1CF03B332C5BA9A10865':'option603',

                    '43E16C3CE356322696D66910FB7914C1':'ipAccessMode4', //第四组
                    '65EE13299F2CEEEB75901CBAD8E3F45A':'ipAddress4',
                    'C39A6F693CF181BE0AD44DF36DF8FFD2':'netmask4',
                    'BF7E382641504029A6B291903A2670D7':'gateway4',
                    'D29DE6143C2DEFA33689D1CDA3696502':'prefix4',
                    '2D6702842AF583DBF206189B1BBC548F':'ipv6Gateway4',
                    'A23FF9F65805DF86936561CA42180251':'ipv6IpAddress4',
                    '03048BB7A11FDD40213FC581E82780DB':'vlanId4',
                    '265F0FBAFE7585E4E3206CABBB74136F':'effectEnable4',
                    'DFA9FDB207A435ACFEF0E20D0CE20912':'option604',

                    'EF2B21E93665D66F99B170EF56E272DA':'ipAccessMode5', //第五组
					'59E8B9A28C96BACA47C83B71A382F5B8':'ipAddress5',
					'D152ACF180E1157C016FD728B998523C':'netmask5',
					'38284C0E4D4F92DADB7EBD6E53FB84A8':'gateway5',
					'237A7C0F89AC43E7260387B3E09455DB':'prefix5',
                    'C511116D04D4991B2E3C37E31A29A88D':'ipv6Gateway5',
                    '63211EE1F3846EC9FAE8BFB404639BF6':'ipv6IpAddress5',
					'7B4D2FF8356EB8311E32334A9C44811D':'vlanId5',
                    'D889E1682C0062DE349F9875D18E0D81':'effectEnable5',
					'FE0721B4D2B2870D02E091F034902CD1':'option605',

					'43DB90E5CB8E1FDB231392A400441061':'ipAccessMode6', //第六组
                    'A897F84DC98C3BF8477CAC4550A55832':'ipAddress6',
                    '3780F5A888F572A8979370A86405E3A3':'netmask6',
                    'C1173FF19D2FCA72A0759B5A50A5AF7D':'gateway6',
                    'D37FFE7F744916E7C7B437DA006FC96D':'prefix6',
                    '107ACEF415420D3764317F6FF0E56C36':'ipv6Gateway6',
                    '0532239B970E70EA49A6D3F65E879B50':'ipv6IpAddress6',
                    '65CFAD5ED74B31A933116B7B4755908F':'vlanId6',
                    '7513B5A49F80E4B1BC11439D025A76AF':'effectEnable6',
                    'ED3671E6F25E5E50F308422580C04D7A':'option606',

                    'CA19B3C115667FB59D17A985752E3DAE':'ipAccessMode7', //第七组
                    '7F35E4949F5B3036DF11DA84A7D8A203':'ipAddress7',
                    '70FDCF6138EDE6A93A2D0D3A2DBD3062':'netmask7',
                    'BE8748EC95F5E7FFBEEC675C22CB78C0':'gateway7',
                    '5FC46851E99E9A183FC741D0B7C2B9C5':'prefix7',
                    'E4455F88F20DB34B67260C20B84A389C':'ipv6Gateway7',
                    '2C1B36E2D13D38C1949B21D7EAAF09E2':'ipv6IpAddress7',
                    '146FE95CCCE85E3A0EA03CC8190343AC':'vlanId7',
                    '0E9FADD67CD88D793AC8E3E35FEE08D1':'effectEnable7',
                    '37063781520D13E7260DECFE533C0393':'option607',

                    '8C20F0B189202C687F5AC9FFB34366EA':'ipAccessMode8', //第八组
                    'AACBC08439506DD53FF14D43D8BBAAE9':'ipAddress8',
                    '0DC6E7845ED7AC49F0028BC6BF70949A':'netmask8',
                    '9AF4006F99BE1F50D6DF63CF6409ADEB':'gateway8',
                    '2FC0691852FCCE8BA5F5DC3864D92ADA':'prefix8',
                    'A9F9177D154C873C7ED2F2B9F4872CCA':'ipv6Gateway8',
                    'D13F3CDB07070E9BBB28C63737B71CD6':'ipv6IpAddress8',
                    '4E1F2EA9F57524DC91DAED7762710D31':'vlanId8',
                    '2049690162B9762C833B3730A4B736FD':'effectEnable8',
                    '24706B310C2A0F177A6F8C9CB41B6B41':'option608',

                    '6D8F1FB40ADD2A9F7679467699A2B613':'ipAccessMode9', //第九组
                    '6BD8E185880D4876A1BD7107244E9AD8':'ipAddress9',
                    'B8C91C04DCF5D380BB4ADDBB83BD97D1':'netmask9',
                    '84167423F6AAC8A3C7DA27C95DA0F94A':'gateway9',
                    '5C49B6D5A4EA2BC6BAB4B10CD52A9031':'prefix9',
                    '40CA021C8B9C701B1C713760A3FDCA45':'ipv6Gateway9',
                    '5B316CC4A5151F59DCD8A32364C959A0':'ipv6IpAddress9',
                    '8F4F683BB2ABF6BD65C0522BC20E72F9':'vlanId9',
                    '20D9809E82944176C7A0F108A06E8BDB':'effectEnable9',
                    '4148029A30F6BEBCD58249BB4FB823FB':'option609',

                    'CF922E683D9358A055BF6E72A7005C94':'ipAccessMode10', //第十组
                    'C7E48586183C32AEA26D881A5478AF92':'ipAddress10',
                    'EF16A044C0C6A3806722AD709A12A3A6':'netmask10',
                    '79BE465D77101FBDE2C21816DC19A625':'gateway10',
                    '412DDEFD19E6E389884BA120E697DD13':'prefix10',
                    'F0AA6573F46C28FB8DDCAC44202F4CF7':'ipv6Gateway10',
                    '04117CAC20EC41900BBD9C13400025DC':'ipv6IpAddress10',
                    '87A9D9BD713A672F3D61762C1E75E2C4':'vlanId10',
                    '3A24CFB1F8AEEC7F86ED6BCA6F9790D9':'effectEnable10',
                    '2DC83EF0FCF722FF2B53949021FE9E0D':'option6010',

                    '5FCA83C51737EC14E201986389E571D7':'ipAccessMode11', //第11组
					'248D5D14083A220973ECD5F28443E375':'ipAddress11',
					'5460A1CCD243E98557F1695EE6152B7A':'netmask11',
					'B9075E0C3FC6F8B17236B1482E0C97CC':'gateway11',
					'DA08242CEBFC4644BE4B88E5F4FE6C3D':'prefix11',
                    '82B8B20D323D9FC1D5634598DE82F260':'ipv6Gateway11',
                    '5C420324F22FA73B8B7039B4539DD6E3':'ipv6IpAddress11',
					'2D945A99D7ABF152DF39E7D52F43426C':'vlanId11',
                    'D839ADEBFE739FACA7D0E7E4FE1DE9D6':'effectEnable11',
					'30169F033486BF26B2C93EB7FD1ADAF5':'option6011',

					'2BD5DD3C4CE952F70739A9EE376B34DD':'ipAccessMode12', //第12组
                    '989202CD4985CF93A046CC71479214CD':'ipAddress12',
                    '3B341793DAE2C2C9D5711817EFF99A7C':'netmask12',
                    '72A4ACC67B4A55A57CE12DCCA0EA08E2':'gateway12',
                    '14AC790790629846A9E5AE63546FA8AA':'prefix12',
                    '52FE1354D9A28D955E0A1E519DF9C8CB':'ipv6Gateway12',
                    '37C3EB0FF956973F248C6A16A5D05F98':'ipv6IpAddress12',
                    'EA6EC41D8217E73751F07EFA11512CA6':'vlanId12',
                    '451DED35B2D9ECF05D862493D0A50853':'effectEnable12',
                    'C51B05866F9D889D1B2B786A9041A8F6':'option6012',

                    //4860
                    '0AA8BB5DC422CD4FBAA167EAD9B1A363':'connectType',
                    'C4C6C5C2D1F0B8C60DE04AF085738DCA': 'linkSpeedNegotiated', 
                    '5B6656D83EB652528BB372466BB6F226':'dnsConfigEnable',
                    'C3E86B2646BE9B753AE5BBD4AE287D7F':'dnsAddress',
                    '96EC51DC6D48BDDD9815BCEBCB7062E0':'dnsAddress2',
                    '88DF507A30A146BBEA9C7E75E4B20403':'accessLMTViaWAN',
                    '08528D7F61F3DCF034F34297C0B8B329':'lanIpAddress',
                    '4EABDBCA4066B02E38EB7AC5AE0FAF5E':'subnetMask',
                 
                    //4860 模式，实际只配置4组
					'F1181A322CBB92B1479E3EAE610333EC':'ipAccessMode1',//第一组
					'0E5653437FA4923D112FB18A4EC5B887':'ipAddress1',
					'37B2B6F4BDD7867D4D88B47ED4C33DD2':'netmask1',					
					'B65581CB85D52BE73E43374D45C3CD09':'gateway1',
					'0052B482DD41C15E817A045449F3EA11':'vlanId1',
					'A8138751D6F88E102CA8FC19E335CC3C':'option601',

                    'F4FB860A695BA34462D3865FD7F35F80':'ipAccessMode2',//第二组
					'75E3A5116241D7B751EFC49D643B00B3':'ipAddress2',
					'B17558B3AE8E6334FD038C7457B20BCD':'netmask2',					
					'A4EFC48D144D7208748DB0D362375529':'gateway2',
					'8BADB4CD731C61B2B3B61ADF3AAE10CD':'vlanId2',
					'699A12F2EE205541EC9A06B3D4B4E71B':'option602',
                    '8D9E2BA50D78A964628296156EE05D6E':'effectEnable2',

                    '7EB8CD1AEC08E5CE6C7D670DE3EE8071':'ipAccessMode3',//第三组
					'7A76826ADD731815034551ED0FF34F45':'ipAddress3',
					'172B14D4FD2E7501E1BD0E260D611EA4':'netmask3',					
					'7A357947000FB13F1E02AB736A6DC0CE':'gateway3',
					'7B6D1933F0F0F4C8736C952DF26D934D':'vlanId3',
					'C5AA7C1F272E02D376905069B46BBD89':'option603',
                    '499954FED3701A2C7A25B73F30F4C66F':'effectEnable3',

                    'E775FD60577CCDF6A99A02545870F5E7':'ipAccessMode4',//第四组
					'58267AB742E1E4E513293DA524E27AD9':'ipAddress4',
					'87ACDEF0B122A790CB03DB26AF3B07AA':'netmask4',					
					'6CC909CFD44131D966D5DA49D833CEDF':'gateway4',
					'2760D10F1620274D3C0F0CE151726E22':'vlanId4',
					'543548E22EB478A90771B8FC8EA57F97':'option604',
                    '927CB413212EED48E3AAD09C1D839535':'effectEnable4',

                    //MLN
                    'CC15F8FA0DB547255782EA4B66A81D41':'connectType',
                    '428FDF654D0B1A2689868149ECEBE33B': 'linkSpeedNegotiated',
                    '0BD81E76854C54757CE8E131EC6B6884':'dnsConfigEnable',
                    '46F91469C5CDE425445F7886095E866F':'dnsAddress',
                    'B29A23EAB895971FEBEA619FE0235C36':'dnsAddress2',
                    '0F57677D79146674188AD7D001FB0755':'accessLMTViaWAN',
                    'C2230768483848BE0767F90EBC479BA2':'lanIpAddress',
                    '051B030E5D76D8B169A8DC6579CD39F1':'subnetMask',
                 
                    //MLN 模式，实际只配置4组
					'99BE4D7952E56DF1242EED6203D8C145':'ipAccessMode1',//第一组
					'4DC95DFD9EDB733F5683BAB8CEB05799':'ipAddress1',
					'91F5692532942FBB3E8E9A7A8323C662':'netmask1',
					'A0DC29B825FE46A630EDFDFAC1368256':'gateway1',
					'A8DD27FB280A1FC5ECD4853A1A6D96C9':'vlanId1',
					'BA919AF9FCD99D10B8AEB3482F2135FE':'option601',

                    '5536DFAEF06C1A1F0F5D692DD689C61E':'ipAccessMode2',//第二组
					'2E9A4DDB87424875DAD9EC90C20C847B':'ipAddress2',
					'B48289296B363C184C5100D63A966588':'netmask2',
					'7657975FB2F214A510145736D99449FF':'gateway2',
					'3D38C95149D911AB9B00AB141EA4CD5B':'vlanId2',
					'0A9960973B62A0DCB2E2A2E2AFF0E4C5':'option602',
                    '5811171CF8C67D6CBB5F62A67E9430F3':'effectEnable2',

                    '435CC20DD289908CCF814D6C7CB19599':'ipAccessMode3',//第三组
					'1563095C7AD906B0331C4581C02A34D1':'ipAddress3',
					'BC89B877CD614AE80A70E070E3165A47':'netmask3',
					'75E66DA3CE63F12870B7705DBE25F24B':'gateway3',
					'9C93B6627334C4794BC9F39C7CDBD243':'vlanId3',
					'F4A95B002D31F8C42F62C7810286F2E6':'option603',
                    '3ACC6A40D03B73A6A7F199C0246B76DF':'effectEnable3',

                    'FA6C2C5A827FE9F655429383E0897F37':'ipAccessMode4',//第四组
					'4613568A959C671703A8FA9383A0BBA9':'ipAddress4',
					'73818323607E91700DE7BCB29A503F56':'netmask4',
					'A3377EFF41FEB9E71BE6D7293FCCB503':'gateway4',
					'5ACB2C23B4A9D8B2B72893A43336ADBF':'vlanId4',
					'28D05495453D5776E409DF14A930CFA6':'option604',
                    '62D14274B1B9AD8A8C74F9859560746D':'effectEnable4',

                    // Static Routing -- BaiBLQ
                    // 第一组
                    '280FDF7FED341072AFDB20B907C2295D': 'routeEffective1',
                    '0685FBF79D10CBEF0BACB54D34A4179E': 'routeNetwork1',
                    '7C3A11ACF938E2645D03439A343404E6': 'routeNetmask1',
                    '9EF0DBC143B19D3283D0328F7DFDC8A0': 'routeGateway1',
                    
                    // 第二组
                    '963BF9C3833235094F51828DB4B0E79D': 'routeEffective2',
                    'F541E50FFE4F99E736A3D7DD88E2316E': 'routeNetwork2',
                    '1324B98EDA2329FDE5FF89FBB9326D76': 'routeNetmask2',
                    'EA988ECB6DCA8C79D9A5ED46759603D5': 'routeGateway2',
                    
                    // 第三组
                    'F047AD70D901CA88037F3702930BC4F1': 'routeEffective3',
                    '1674AA9F785F4FBB4D709225F9CB88B5': 'routeNetwork3',
                    '484E64D7B7AE1D30674674DB05A8049D': 'routeNetmask3',
                    'F5F521C29F61C60818C43D6D0E74AB01': 'routeGateway3',
                    
                    // 第四组
                    'ED7A0CDCDC9ADA3EEE7965457DE9706D': 'routeEffective4',
                    '03640D662CC7E785135A5B28DE18B6D2': 'routeNetwork4',
                    '0DEAB2AF996537961B0AE8F8623A4DF3': 'routeNetmask4',
                    '73975ADCD8F459E1FCE662BD7F2896EB': 'routeGateway4',
                    
                    // 第五组
                    '506FF0A7C98BBF9368EA869D0C1DFB18': 'routeEffective5',
                    '17CF91D30E28A0169F658BAB0E8E7777': 'routeNetwork5',
                    '25F7394AFA3C3FFD27BA2AEFC4A75E71': 'routeNetmask5',
                    '33337F7601AE11F27CDEB5EB6DEFE2F0': 'routeGateway5',
                    
                    // 第六组
                    'CB870395253C59FD8792AC927B5C58CE': 'routeEffective6',
                    '1B3ADBF830175E7284B53B0E87E447D7': 'routeNetwork6',
                    '965A428C7B6FDD1A879FDC87FA29686B': 'routeNetmask6',
                    '33771243414A7C8CB4F5E74FFC752971': 'routeGateway6',
                    
                    // 第七组
                    'AEA89A52AE0458059D009EB933F18978': 'routeEffective7',
                    'D08F26FDB471F156114CFF1AAC2340F4': 'routeNetwork7',
                    'C4FA11C07D63BB5FA48A3EE41781E5D6': 'routeNetmask7',
                    '1923A54275C928A912F1FCE5D8126199': 'routeGateway7',
                    
                    // 第八组
                    '2A5AD488BF59BC140F0C092137A179FB': 'routeEffective8',
                    '3FE2D073CF05F6513CCC9FA94B733521': 'routeNetwork8',
                    '012498E6A5FF039D11A5DAB9501801B0': 'routeNetmask8',
                    '9FDBCE56D9C11FCBB6993EF4D0EC032C': 'routeGateway8',
                    
                    // 第九组
                    '78D784F7D0213158E564C849B4365587': 'routeEffective9',
                    '3DE7564722E3CAC2F0421F06BA3CFB23': 'routeNetwork9',
                    'E283C2D739EDED6531BED3F4CE736614': 'routeNetmask9',
                    'A01B5197437C832D8ECC48B3B92AB015': 'routeGateway9',
                    
                    // 第十组
                    '0070CF76DF7C491D84AFF142F3BCBCEC': 'routeEffective10',
                    '56F66F0F8213718EF9F3A93233CB4DD3': 'routeNetwork10',
                    '60926ED8094A64EAB5303B4EC9410B98': 'routeNetmask10',
                    '9A5D7CFEE23270A6B43E74EE4F0DDD07': 'routeGateway10',
                    
                    // 第十一组
                    '9F95423667DD65DA5FC481F8D4B61951': 'routeEffective11',
                    'F2989E491C43BE9FAA590CD8B971AD3F': 'routeNetwork11',
                    '87B2972AD721020F85B88D0509161BE2': 'routeNetmask11',
                    'CF99F53DC3DC61A503ECDFC1329185CA': 'routeGateway11',
                    
                    // 第十二组
                    '2E90C3A4CE9331439598986974CBFA37': 'routeEffective12',
                    'ABEA7043D1C0705F229C92E82804E37E': 'routeNetwork12',
                    'C059C57D06A1B99190C516EC78F9AD90': 'routeNetmask12',
                    '4F5886EC9C50BD0F4A7FD90F29355306': 'routeGateway12',

                    // Static Routing -- 4860
                    // 第一组
                    '7659FE02E31FCEFD878CB3D64F6844B8': 'routeEffective1',
                    '967775993834238B1F12DD0B407C31FA': 'routeNetwork1',
                    'DBDDD86920866434385D17AF640DF4FB': 'routeNetmask1',
                    'F50D3F51E24DD724A364884E748C14AA': 'routeGateway1',
                    
                    // 第二组
                    'B5539AAA70AB2F30520BCCDE2EF38810': 'routeEffective2',
                    '519EAE8F15B8E47541F2C494595CE79F': 'routeNetwork2',
                    '5E15A0FC84DE21A513CDD62304A0EFD6': 'routeNetmask2',
                    '290F39542FC1D81415522756E9E81FD0': 'routeGateway2',
                    
                    // 第三组
                    '90D8F313F04035D8F588732C15316ECF': 'routeEffective3',
                    'AD9DE2020402E49D2067FD032943A817': 'routeNetwork3',
                    '0B73922FE92F3D3E8C07421B61F9956E': 'routeNetmask3',
                    '97C486A099F2460DE5B25698F1AF3F8B': 'routeGateway3',
                    
                    // 第四组
                    '735DCE8FEAFA01A1FF951065788F24BC': 'routeEffective4',
                    '742715D452E791E8AC9F1481C8475EDD': 'routeNetwork4',
                    '3A1D12CAC38CD6CC5EC94967B4F97E3E': 'routeNetmask4',
                    '8C44B8C4A5A04C33EC41F78265C672E7': 'routeGateway4',
                    /*
                    // 第五组
                    '5AAF0A11862CBEFB26B52DC2050BDFDD': 'routeEffective5',
                    '5FED72BAB1B2E3A25394AA4CCED78BD6': 'routeNetwork5',
                    '6476C74472E5F960A933695041608A4D': 'routeNetmask5',
                    '92BCF20A8A8AC6D0B396874AF3D369F8': 'routeGateway5',
                    
                    // 第六组
                    'F10954C7DB8F19B25CE9E647D4C93826': 'routeEffective6',
                    'EE99E136F11E128ADBD3B5FD493E13C6': 'routeNetwork6',
                    'D0AEF6F96A42FB03DB3B9DCAB440818E': 'routeNetmask6',
                    '7ED713527B65A2E4299D69239396F144': 'routeGateway6',
                    
                    // 第七组
                    '90F5AF90D6BE154212CE91069F4A8113': 'routeEffective7',
                    'D513E46C6F82A5B2946B337D9B588B56': 'routeNetwork7',
                    '0EBA48AB6525DA9C704D017AC152B115': 'routeNetmask7',
                    'FF9A793EE2D7DF920DBF57ABD0BF71DE': 'routeGateway7',
                    
                    // 第八组
                    'C65F186FE2A7FB8FBB45074973CF675F': 'routeEffective8',
                    '0C32FD061A06E1891E7A4E659454DB26': 'routeNetwork8',
                    '8EEA7453AFE3EBAB66A211833F527D2F': 'routeNetmask8',
                    '7DCBD7D7B5C263EA40DB7A32FA7D4F2E': 'routeGateway8',
                    
                    // 第九组
                    '1EA08D3F8E38C8AB57568C287E68819B': 'routeEffective9',
                    'DC04732B833A08C9E9EE97B33C8FB794': 'routeNetwork9',
                    'F2059182F8476FF7892CA18BF73EE99A': 'routeNetmask9',
                    '6C5D0319C47C0FEFE40B622AABD4454F': 'routeGateway9',
                    
                    // 第十组
                    '8634FB77A33B4D0498FE229E108EE2DE': 'routeEffective10',
                    'F1358EE74EB12670CC2177F4AFB8BA8E': 'routeNetwork10',
                    'C57963CE222DE17A6DAA8C0A2E0DB554': 'routeNetmask10',
                    '557A5DDFA58EFA69CBC58343457C60D0': 'routeGateway10',
                    
                    // 第十一组
                    'FAD2563E99BFB2089D9079458F185CCF': 'routeEffective11',
                    'EBB00F10108C21AD93F10D4509755E2E': 'routeNetwork11',
                    'DC56A81B0DEDEF177B2A09545146F601': 'routeNetmask11',
                    '63181F50FBD56E2735FDCFE0C2E648F2': 'routeGateway11',
                    
                    // 第十二组
                    '142989DFCDBF09DAA748F565380A31A1': 'routeEffective12',
                    '0B79D650239ECA87EFCB612BF93B2E0F': 'routeNetwork12',
                    '16FC03E775843F089716D79BC925BCD5': 'routeNetmask12',
                    '581289AE611159ED546D83E0D81F16D1': 'routeGateway12',
                    */

                    // Static Routing -- MLN
                    // 第一组
                    '36E5EBBC81EAE74FD8C77510B028E3D6': 'routeEffective1',
                    '6A4E9366194893E75E7E914B8D6BC889': 'routeNetwork1',
                    '572C6F0DF86176E7617E35D81FAA8FA0': 'routeNetmask1',
                    '470455C290BC8162EFB8E5B30AC0B834': 'routeGateway1',

                    // 第二组
                    '8727EA222D483BFA8C8ACE89FDAE4278': 'routeEffective2',
                    'DDD954761C10E86585864E19505C0BCD': 'routeNetwork2',
                    'CD4CE1BC8033AC4A6FB050C6DFAE612F': 'routeNetmask2',
                    '8BBCC9D1E4A3FE0B766F9EB1B872BDBB': 'routeGateway2',

                    // 第三组
                    '22AB9F318136F33A41FA5B36F39751DA': 'routeEffective3',
                    '4E802FDED7CA01EDEAEDC2CEEF43C848': 'routeNetwork3',
                    '946897DFD4819D288373AC4CC75A5498': 'routeNetmask3',
                    '92E3052C4F8CF54269C849D98FC9480E': 'routeGateway3',

                    // 第四组
                    '8A83C76F45312772844F24FCECEDF265': 'routeEffective4',
                    '40D4492ED90DE43E4BC058F8A0C7048D': 'routeNetwork4',
                    '31B4C8692EF8ACC671BC53AB51D745B4': 'routeNetmask4',
                    '7FE652C93A587B2C86F07050D24ED2EA': 'routeGateway4',
                    /*
                    // 第五组
                    '84CA2A82EEB48705518758E20FC655EC': 'routeEffective5',
                    'EA38FC3799F21225D55F537395578BED': 'routeNetwork5',
                    'C028FF56529C4B54023003CFD6E36828': 'routeNetmask5',
                    '4976BA298838B83D40A7D10C7F444E03': 'routeGateway5',

                    // 第六组
                    '641E6461C0D4F30A58A8FB65C0574F07': 'routeEffective6',
                    'D665932CFA6BF655D7447B394E92D7E2': 'routeNetwork6',
                    'DC8839D592301C073567707CD51172C5': 'routeNetmask6',
                    'B7A8E59A98A2CC4ABFBE8D9509991573': 'routeGateway6',

                    // 第七组
                    '2F08A59213236A0A2F0A6173A49CD685': 'routeEffective7',
                    '35A382CD69EB60E919295374229A8D21': 'routeNetwork7',
                    '4A66FB92FC1E0D3E299C6F948026A3DE': 'routeNetmask7',
                    '7FED585DBB726B89741158913FC97947': 'routeGateway7',

                    // 第八组
                    'CDF26CE0B39D9E6D9957A8932BBD52B0': 'routeEffective8',
                    'AAD7F033157543EF00AD916D94916BA1': 'routeNetwork8',
                    '35758A0CB51CD7F8147DED64A5579687': 'routeNetmask8',
                    '86DD23F5E61E3F5E0BBA1E31F00DE137': 'routeGateway8',

                    // 第九组
                    'AB0CDE657793D509615D2C4787A25178': 'routeEffective9',
                    'AF51DF4DE07126DFDEFA785C4C31F5CA': 'routeNetwork9',
                    '0A18AF27CD577976C1DA554BD35DEADC': 'routeNetmask9',
                    '42027D677D4EB6C220A2A73E8F583338': 'routeGateway9',

                    // 第十组
                    '76907D0449DA6A2F52B4434AFF962057': 'routeEffective10',
                    'D3712A68435A729838205477FE5ED82C': 'routeNetwork10',
                    'DF93E2F375C9D46C72C5BC0ADFB16DB2': 'routeNetmask10',
                    '2786C003208B3343E36EB1FD411B9BF8': 'routeGateway10',

                    // 第十一组
                    '4FC0A260F93B62D53A232F030F5EFFB4': 'routeEffective11',
                    'C5854226A69078A09F2A990626F690A0': 'routeNetwork11',
                    '8ADC6473E1EAEC86402B16534CCF858D': 'routeNetmask11',
                    'C3A3E39D5B0A3B0EA2A0350153E2D92E': 'routeGateway11',

                    // 第十二组
                    '5A3390E3FBFF8C677BE6CE76173B51E5': 'routeEffective12',
                    '0E914052B6C0AEB9D41805375911B3C7': 'routeNetwork12',
                    '8F15F04DA074248899A81297447492D7': 'routeNetmask12',
                    '7DEA6F14E28DB98BDE2BCD0F36DD4147': 'routeGateway12',
                    */

                    // O6 
                    '43E9C11A9F368DE119EFC8E932A3D38B':'ipsecEnable',
					'74D948F48011467C6B4B3FE8DA30B6AC':'ipsecList',
					'B5C560B313A7BABE38E79E964F3184EC':'index',
					'C7F9E13B09654EC3D372CDAB1D4B03BB':'configEnable',
					'ABDE5B7D8E72DFCAC18DD3D72A3F5032':'leftAuth',
					'0D139FD289818C3CC770BC46CF74D216':'rightAuth',
					'DFBC32301433E1ABC6B452CE9D594D4D':'gateway',
					'3171B9B0109D8C75FDC71E89955FAAB0':'rightSubnet',
					'155B0D6FBF883259ABCAF107A2AC62A8':'leftId',
					'322BE625531A51CDDBCEC2E291BE3D02':'rightId',
					'EA14360EF34D772CD4774B236E8528A5':'leftCert',
					'535F9838514773A49966BDF750652FB9':'secretKey',
					'0CC6BE7099C3B98256AF9D064F5DEBA9':'leftSourceIp',
					'AAC62CF372BE036B53F4C07066985F64':'leftSubnet', 
					'6BD00E56FA9B6E976510B7FE3D34B295':'ikeEncryption',
					'E4A44BFC79A3B0A807CDC11720409393':'ikeDhGroup',
					'07A3C1CD78E5D9F23A0819165339D01E':'ikeAuthentication',
					'7274C98C5BE9B6D5092DB7AEAD7FE207':'espEncryption',
					'6E5FCEE222D7F76BC168A2D4A8E96D24':'espDhGroup',
					'755BF9170A2088308C0FCB0BD3BBDC15':'espAuthentication',
					'42BA70759B0D63310364481C9DFAD2D1':'fragmentation',
					'D789576348A1356675C5C2A5DA18DAAE':'keylife',
					'A2B44779597461A5AA6DC209A8B1AD49':'IKELifeTime',
					'E0C3D5460C86B33B765AA6CE2E9CFD94':'RekeyMargin',
					'5BFD08C07F599ED205A8CB672B156495':'Dpdaction',
					'B1E823BD3980ECEF4D6E5F1FB044527B':'Dpddelay',

                    '1CB50F1D946AF14B64BDE37501B46768': 'mtu',
                    '1D942332CD5F7DA0B9D7A3F6878065E1': 'dnsAddress',
                    '7410F382228F12A5665DDC5687D01F6E': 'connectType',
                    '58B51F880A0E9E353AC06276B261F162': 'dnsAddress',
                    '1D942332CD5F7DA0B9D7A3F6878065E1': 'dnsAddress2',
                    'B5AE345E76024B92046DB9A1F2D82AF6': 'accessLMTViaWAN',
                    'A4E3C35055EBB35B2EFA0ECCAC18FFE7': 'quickInterfaceBinding',
                    'E1033BE78BA3C3D01CDA538853046C46': 'subMachineAddress',
                    '3F256578F7F52A65742123D3EAA49ECC': 'lanIpAddress',
                    'E0A111179A09F1DE1516D08F2A01E96F': 'subnetMask',
                    '621BD8BE5FA1C1BE22FA36240804B0AA': 'slaveInterface',

                    '43E9C11A9F368DE119EFC8E932A3D38B': 'ipsecEnable',
                    '74D948F48011467C6B4B3FE8DA30B6AC': 'ipsecList',

                    // wan config
                    'DA9964DEB6A2DD02C182576C5EC7C8EF': 'gateway1',
                    '513105C3B0EF5907E6044992F2B74283': 'ipAddress1',
                    '313DEB1F632E31F04AA7E11CA1D9DD29': 'ipAccessMode1',
                    '60E5FEEAC130D1857BE8098C078BD33C': 'netmask1',
                    'E31C98B623EB8E1AFDD439F156948A07': 'option601',
                    '724BC61E5569B95E554555E33A22DFE9': 'ipv6Gateway1',
                    'D528CB238447D7481623D4BAA5C337E9': 'ipv6IpAddress1',
                    'C3A89E96AFAA04693EA6C5403E516163': 'prefix1',
                    'E7F82AC5177E383AF1A41132DC766C0A': 'vlanId1',

                    'EBDB1E5B725B0F8FACB2E262AB7BE978': 'gateway2',
                    'E126E72A9A23A6CBF5BDCDEF4C3AE266': 'effectEnable2',
                    '36E59566AAA1E6A98B815F4DCD56E17C': 'ipAddress2',
                    '1BCF8B6D1508AD0A15B514EA18935C12': 'ipAccessMode2',
                    '685C9DEEDFA2585A8C3BD142CCF7E595': 'netmask2',
                    '24AFAF04BB0DEA15FBF69C73D0DC7ED6': 'option602',
                    'B9E2D86F78A903610F6F90014E589029': 'ipv6Gateway2',
                    '79DEC984AF0DF5552E0BD5236EA2E6A6': 'ipv6IpAddress2',
                    '897E9FF2D835FB973BB413DB01E9A2B1': 'prefix2',
                    'DFDD341C52F8D88812C2F67A80E13D36': 'vlanId2',

                    'BF8D7128CEDEF5EBCBE7D898113BF2F8': 'gateway3',
                    '20CDB71B22F96CEDFC81B3BE58517C52': 'effectEnable3',
                    'B3B22188A4C8E7C9BB86F1C642476B1D': 'ipAddress3',
                    '92AD2989FF73912EFEC6EAD631E2B896': 'ipAccessMode3',
                    '1E4FE5226CE750E0DF5DEE9ECB8F6EF5': 'netmask3',
                    '981174BCF39D1AF58F9DDCFE6416E47B': 'option603',
                    'D851F3ABC2631F9226C11B56A9A5A02E': 'ipv6Gateway3',
                    '9CCC97C4470C7E238CCF1CF166F81C90': 'ipv6IpAddress3',
                    'F7D00FDE58E288A8FE2A01FA45846094': 'prefix3',
                    'D515A38F73B35F92D5A76F2416AC9DA9': 'vlanId3',

                    '574F50B1CB9AD0808DB0CD1457EA5986': 'gateway4',
                    '3F812BDF50B5DDDAE15E2D5EFF488158': 'effectEnable4',
                    'C8C7CC6511F0C48EBE4B8F94DB99478A': 'ipAddress4',
                    '5EA6F1B340C9886F0E907A8E39567DFC': 'ipAccessMode4',
                    '30F579281E766D117CE06D36D216A928': 'netmask4',
                    'A9525B9023B9A88C3222DF1A05D5C092': 'option604',
                    '890B7917FD5C6CC639ECCBD9B99A33BA': 'ipv6Gateway4',
                    '6C19292D3CEA01FAAD327559B2CA2027': 'ipv6IpAddress4',
                    'D45E6C911E3BD2D74E3F2F157FB04DDC': 'prefix4',
                    '5732E8CFFD512C98B7B94C85AA19168A': 'vlanId4',

                    'F5E45466BEC2F594660D320811D6510B': 'gateway5',
                    '7E980C3A746EB139F96FA4E467AFD935': 'effectEnable5',
                    '19F83907545A449109D268559B6CD322': 'ipAddress5',
                    '860A514EBDC9F5CFC11B2D6F5A1FE8DC': 'ipAccessMode5',
                    'ECFDFB4BF5F02E961839A163320826A9': 'netmask5',
                    '899573B44D01B5B158D9165A65D6DB06': 'option605',
                    '847CBFF2E8F94347A7AC771F74EE53A3': 'ipv6Gateway5',
                    'BEA4B7470F9A39B88F8F94BB0E0B0268': 'ipv6IpAddress5',
                    '2F725EB7BE5F0B83B203BC67B6101CD6': 'prefix5',
                    '95815FE7092E0B1ECC2E9E56238548FF': 'vlanId5',

                    '460F941E9FA8E7CB6C0D2DA33F2589F0': 'gateway6',
                    'A905F0B52136F43E81546F1D96775868': 'effectEnable6',
                    '6D38ACF244A57F6E8EDE6514D19F0977': 'ipAddress6',
                    '693E3214AAD223E37BB5264641442A32': 'ipAccessMode6',
                    '9F2ED532EA88A21FB8B2AA16A636301A': 'netmask6',
                    '8323A0F14D75E2AD393B4DF16AC1A9CD': 'option606',
                    '3CBC5F87C5BE382F942CC7DFD88F6539': 'ipv6Gateway6',
                    'C091664750E0CC6105503DDCCFDBB706': 'ipv6IpAddress6',
                    '746F797EE6AA9147FB58429233C6ACF7': 'prefix6',
                    '2E34928AD5226C97F31E84E4B5027E89': 'vlanId6',

                    '428FAF67CF2715C81ED2D0F741B689A5': 'gateway7',
                    '92F56EEE3BD0D7EC7C96331E8F6BB33A': 'effectEnable7',
                    '42269300C23FF85C0F7195634F800264': 'ipAddress7',
                    '6F8CB966C7ACC47EEC1BEBAD60A719CD': 'ipAccessMode7',
                    '13F73D654B6AE7F6351085560383053D': 'netmask7',
                    '48C0DD808786ECE8193BA8C3BA511095': 'option607',
                    '3B0B2A92421B656DDAD5AF10496117CC': 'ipv6Gateway7',
                    '8338ADBB6AEE23A4681B8766A3871465': 'ipv6IpAddress7',
                    '4CD179BC857692EBB4A8B51DC2D9284D': 'prefix7',
                    'DDF00652E81A3FC272DF1ED85A470FE1': 'vlanId7',

                    'F225F0507F90A082527573DAC06D083D': 'gateway8',
                    'F0EF7A08EBD839F35201237F743E456D': 'effectEnable8',
                    'E708329885190CC9F98E66CEF76FB0ED': 'ipAddress8',
                    'C952ABC50067691AFC6ABB9D0DE7300D': 'ipAccessMode8',
                    'BCC5E118859DC0796D7E01E9E97CA793': 'netmask8',
                    '731807E97860431A43AD05F47DA45BD7': 'option608',
                    '0A38B652CFE60E431A3FB32910F3AE39': 'ipv6Gateway8',
                    '7C997492ED60827C6827AA8868CE75B1': 'ipv6IpAddress8',
                    '0AF37FE3D4D1BB23417D753F3C10D15A': 'prefix8',
                    '61973B228BC3134F27DEBBBCF9463E82': 'vlanId8',

                    '75064E82A4EEF14E721915911CE195AF': 'gateway9',
                    'A959D8F21EC67995F882DD43E7EAF2AF': 'effectEnable9',
                    '022EE5DDB0429652B68630DF90DF0AD6': 'ipAddress9',
                    '4D5C0F35BD66AAD2D0446CDE98865953': 'ipAccessMode9',
                    '252DF1C1E02C369FDA4DF6CC4F2A3B76': 'netmask9',
                    '279C36FEC774C9DA49742A1E44312211': 'option609',
                    'C91C7C81F6A462B6DFE18673FBB345D0': 'ipv6Gateway9',
                    '8D6FBB9B59673C93335A3750B6527451': 'ipv6IpAddress9',
                    '2ED893FD1AC28A8DA605103A3643E35C': 'prefix9',
                    'DFFB72655BA6EB532838206C8CA79D37': 'vlanId9',

                    '9D4C5673F1A15AAD5A29B074AE1005AA': 'gateway10',
                    '6594D2BBBC7CD2E5B8A6614A7D92E7EC': 'effectEnable10',
                    '7F30D276D773768A9F4CDD02BB36130F': 'ipAddress10',
                    '20F685B53B0B2FD3EBD9CD6DA6A68AF5': 'ipAccessMode10',
                    '838682ED8F4BC8E2FD8CB98004FFC76B': 'netmask10',
                    'FA91394C8FB1854782094353D6FED4DB': 'option6010',
                    '4A501CF8B0F441872BB549C69648C4AD': 'ipv6Gateway10',
                    'A293CA059F6F4B7AF90D1984AD77AC36': 'ipv6IpAddress10',
                    '179135C06CC04191C01B20EBF5C9763F': 'prefix10',
                    '235884535C5CD2A67A4524471B3FCDBA': 'vlanId10',

                    '4BB9898FB0A48309BF75D64DA527F3E7': 'gateway11',
                    '7097E07ABAB3BE536753F9637E672EBF': 'effectEnable11',
                    'CFA8664E80BCCBFEDBBA34E9E824E78C': 'ipAddress11',
                    '558D5846B7D616420F6EC9B81164EFB3': 'ipAccessMode11',
                    'B16FACEA459E2C7A720C6AF6814F402A': 'netmask11',
                    '68055A3D4109AE15B81E291E7F19EC71': 'option6011',
                    '334A094B01D7CA406D6450335D186EBA': 'ipv6Gateway11',
                    'E8DA7B7DE66EC5583813F2B1352BAF6A': 'ipv6IpAddress11',
                    'ABFCA4230A6E0E889150A110F6E583FC': 'prefix11',
                    'D1110FE5DFDB8F65CF052AA991888935': 'vlanId11',

                    'CC93E9537942826ADCD790F3E8928739': 'gateway12',
                    '9F4BFE9F564365654B1837211B05764A': 'effectEnable12',
                    '28F9913681A7155AD140819FEE6FFFE4': 'ipAddress12',
                    '9D44AB8DAD71B07A9021C13276CC95E0': 'ipAccessMode12',
                    '0298333A152D17EFDC6381CF632AA77B': 'netmask12',
                    '3451A8DFCD17952C4BB131E710E5E2D4': 'option6012',
                    'E5262D747882AD63605EFF0B7F793C7B': 'ipv6Gateway12',
                    '92B4C6A5A2EBD6545A94CF1CFBC9C0DC': 'ipv6IpAddress12',
                    '5537C6FA9A962D7FB8077BBC7CAC0AAB': 'prefix12',
                    'EE5DD7D11DF9D03FD1EE3F85011D8682': 'vlanId12',
                    // Static Route
                    '8ED80FF0E715017D731B3A12E2E1150A': 'routeGateway1',
                    'B709B4B718B50818341DBC20C38ECC28': 'routeNetwork1',
                    '2E4F2C6F22960F6C7047A9184C52CAB9': 'routeNetmask1',
                    '00159FE2766B18AFBB1429C5A1647378': 'routeEffective1',
                    
                    '72119E271A4F77DEE3E7A86F470251D0': 'routeGateway2',
                    '4F780E922EF8DE20D26124F13E3143A1': 'routeNetwork2',
                    'F5F1C99D5891E7B02B1F52645F87F7ED': 'routeNetmask2',
                    '509D2124B54E542FC8E13AFFEC85C6AC': 'routeEffective2',

                    '79B2D812B2C42B71C83102212E36E4CD': 'routeGateway3',
                    '1B6DCF7DEC80AF8E99D381BA8D599A1C': 'routeNetwork3',
                    '33765508DDECC8F114785CC65B13DCE4': 'routeNetmask3',
                    'DBE258C2C70E3BAC01312837ACD2B30B': 'routeEffective3',
                    
                    '56DA10F40537A6DF4960743B6747FD71': 'routeGateway4',
                    '2A1454F4AF9E8CE10957718C01AEE087': 'routeNetwork4',
                    'E22FD23BAD756B7972895CA7810F03C4': 'routeNetmask4',
                    '6BEBC91BD0CF7879C5B558819E0518CA': 'routeEffective4',
                    
                    'E7FCB63806A83B1982ACEF4D83C75C21': 'routeGateway5',
                    '652568BD2CDCDB2E9553871E107242E5': 'routeNetwork5',
                    '56B00626D8D99E74E7D92952C8495F11': 'routeNetmask5',
                    'EB306D4826487A669BE169C3C0BB3ADE': 'routeEffective5',
                    
                    '524AFCF5E0A65DC6C4D87AA323F5278C': 'routeGateway6',
                    'F948CB174A90A2D7848A529D4CCCA7DD': 'routeNetwork6',
                    '3E2BC923DD621C310F073242FB09A821': 'routeNetmask6',
                    '541BADA198DFF873B58A77AE24B15AC4': 'routeEffective6',
                    
                    '0C2B204994A1DE79CEB36FA50BA22FEC': 'routeGateway7',
                    'B2D655DFD9F54D9F15F3DE950DF301A1': 'routeNetwork7',
                    '8F4D144FDC427979826AC94E66EB8793': 'routeNetmask7',
                    'F670CBCA5A0994A4F8D7CA39781D1A12': 'routeEffective7',
                    
                    '9E2611DC4067E61CA5121C77F4A01683': 'routeGateway8',
                    '41079C745F438CF878EDD51E4CEE4D70': 'routeNetwork8',
                    'C11FE390685504920BC4C82E602F4002': 'routeNetmask8',
                    '49C60881ADD41287AC202F8B57D7947F': 'routeEffective8',
                    
                    '2EBC15A848EBD8553EDC19F777AA7135': 'routeGateway9',
                    '18989FC233BDFF246F82FB7C4A76B23A': 'routeNetwork9',
                    '01B080B9E4B71B912B1DC8D2C52749F2': 'routeNetmask9',
                    '2745C27F1CA0D3FBA37765C7ACB2BFA1': 'routeEffective9',
                    
                    '07DBBD4FD18014A9FD0BC578D85B6FE5': 'routeGateway10',
                    'F8FC4DCA887C8BD35170750DCEE96E7D': 'routeNetwork10',
                    'E64067D6E6D0E8AB485E9C498D5A5F39': 'routeNetmask10',
                    'F7E78EF3681432AEB48C6F063C618794': 'routeEffective10',
                    
                    'BF92801FB888D1C7209B285F9D1BDE75': 'routeGateway11',
                    'EFB6B81A438C8CF9DD1E190F57CDE7C7': 'routeNetwork11',
                    '64BEBFE942EF1D8FE5114197BC1B7118': 'routeNetmask11',
                    '8D587D1E7B67A9DCFB93423CE66FB3B5': 'routeEffective11',
                    
                    'B82AFAA7CFC10885CFB61DCB3C5E31BB': 'routeGateway12',
                    '5CFCF77A6348F546204744BC7756C5CF': 'routeNetwork12',
                    '8C0ADA75E46146E79BCD6550A73E8EFD': 'routeNetmask12',
                    '132A513BF4F705076C978C39CAFBFDFB': 'routeEffective12',

                    // BLQ BLN MLQ
                    'DC87979FB32B573741D90CAEB710D9DB':'ipsecEnable',
					'74D948F48011467C6B4B3FE8DA30B6AC':'ipsecList',
					'36BB6ABD0A241F1B4EF505CAFE672770':'index',
                    'A0F003846871381E472526DC8655553F':'leftIp',     // 新增的？
                    '7EB2711EDC49771D26E001655E073DC0':'tunnelName', // 新增的？
					'23046FED944BC29634B3EDD21DD8BBD1':'configEnable',
					'7848C1154074EAA7FB0587A674EFE9DF':'leftAuth',
					'2572FFBFAE75C9A817AEDACD1E2ED771':'rightAuth',
					'E20D789531CE597B48CEAF62983FD2EF':'gateway',
					'65C6A2646C166ECA26A67189D54CFC6D':'rightSubnet',
					'BFBAF2C310E126D235D00C8E76EAD9B8':'leftId',
					'E5779C1E7E93B8955C9C27F8E0BDA19E':'rightId',
					'4DD8C583186DFA9EBBCBE06C846C010D':'leftCert',
					'3D8B66F9E085C36E7D5A36C7BFE268A4':'secretKey',
					'4A82B81341A423A44526934936F03C66':'leftSourceIp',
					'04AB87EF4F8E53374163D4CFB23D9E4F':'leftSubnet', 
					'BFEA45776E33694C86BF906B6820160B':'ikeEncryption',
					'BA1FC20893A5DDCC15D2A74AA6278675':'ikeDhGroup',
					'D54F4FB46F958327CF296F134E751F0D':'ikeAuthentication',
					'C3A4637CC79476A561E8DACFEB82F121':'espEncryption',
					'810270A2E39932A9B9C7D022DF4F4FAC':'espDhGroup',
					'E320507B203781340AD85DEA3C2DBF19':'espAuthentication',
					'757BCA6EFD79BF1C01C6128A8570E0B8':'fragmentation',
					'69CC45788B56D2BC97BBD895B1433F5F':'keylife',
					'06FA8CEF20D41C571FBA559B4D2ABB71':'IKELifeTime',
					'9B7512272A1C16E4DCCA792280B3B7CB':'RekeyMargin',
					'45407C34C737BF639E9EC6F19F95019A':'Dpdaction',
					'5D6311B661E1BD1CEC33E2DBEC50D771':'Dpddelay',

                    // baiblq
					'74D948F48011467C6B4B3FE8DA30B6AC':'ipsecList',
					'897FAEC453F6DC1C02CB3A08A41D9741':'index',
                    '7DFB623CB26C257B3E8E84B12F532CF7':'leftIp',     // 新增的？
                    '6425C8285243C5DD973AF4909E2BDA51':'tunnelName', // 新增的？
					'37B87EF80E0CFA59CFD03E0E97D7ADC5':'configEnable',
					'37ABDE9D47E0467E5BBB4B9C6CB97B41':'leftAuth',
					'C879A6EDA6C20391DA08B31388325777':'rightAuth',
					'0F6671FF119147188BE1494761F5A3D4':'gateway',
					'8535E98738442F4BD834DD06C06DBAE3':'rightSubnet',
					'A90A43B2CF455AD61BB6E473C45C34BF':'leftId',
					'E4F4F96CC5ACEA820C50305CEA5CDAF1':'rightId',
					'6A787A022BC5AD7BAACFEF6D7545F1A9':'leftCert',
					'BF602883A22128EA850F3B1988D5D665':'secretKey',
					'30EE4618825FC23F427697D45AA37700':'leftSourceIp',
					'DD350703C97A3F6C9C4C849222444306':'leftSubnet', 
					'BD67883E68CEA91959EB22DDFAF946B3':'ikeEncryption',
					'91EE44DD5ED60E5DEE7FBF897AA0D5A9':'ikeDhGroup',
					'F56B328988E06670A7A0D9865A4E5067':'ikeAuthentication',
					'1988027F63304D7970D416B59DB42620':'espEncryption',
					'A15D5E3B6CD322A20646686EC4EBAACF':'espDhGroup',
					'191796A7D0997D014997CF017687AC12':'espAuthentication',
					'399F43CBFECD0DAF6E8C75379C14641D':'fragmentation',
					'0F311C26B86618A7CCE14C52A4BACB8C':'keylife',
					'02A64DE5AFD1956599F8CD3DE1F7DC69':'IKELifeTime',
					'5AAC5198E11ABAFC487D959A4653CBF7':'RekeyMargin',
					'72828244559728CB2DF8176BBF374CD4':'Dpdaction',
					'22E5666674ADC8F3ED9387FD9348508C':'Dpddelay',
                    // MLQ
                    'A0F003846871381E472526DC8655553F':'leftIp',

                    // MLQ wan config
                    'A5B3053307A2A42BE5628EBF85C86ACA': 'gateway1',
                    'A035A5FF54E059B30BBBDA066A8D8C5C': 'ipAddress1',
                    '5F6C249090470B3E82ADD70A2D033CF6': 'ipAccessMode1',
                    '404477030027AA44808FB0E0F99CDD2F': 'netmask1',
                    '0D35D8BB3813698C45FE27E35218E11A': 'option601',
                    '8B573B85A231814FC1E14794017E37E8': 'ipv6Gateway1',
                    '4D27D609AFA9491F787E5BE064D810B9': 'ipv6IpAddress1',
                    '92C21E42E2E65BE8479379E3D87C30EA': 'prefix1',
                    '72CCD4620F337E287E292E1ABD9680DC': 'vlanId1',

                    'CCCA4642112E68BBC5F0AFB3464352FD': 'gateway2',
                    'F834EE161411B20F5CA1D52BAE9880A2': 'effectEnable2',
                    'DFB05A7A504CD9484126EDC572833BCB': 'ipAddress2',
                    'B4D9A71EBC1CD62CDF73E902A6887275': 'ipAccessMode2',
                    '30C33E767D1BA93A80176D8DE22D911D': 'netmask2',
                    '11154E82B3D686A8F83337513ED9A066': 'option602',
                    '9BEC27C60D889E98A99932E729E85814': 'ipv6Gateway2',
                    'E44287D7B17A10B38823B3B1DB386AC9': 'ipv6IpAddress2',
                    'B0CB85D82C076C579956FEEA6CD5C3AD': 'prefix2',
                    'A6E89C4E96A63C6A26D82AA69B57F265': 'vlanId2',

                    '58B31BCF4381204A31E57530CF983159': 'gateway3',
                    'DF74D6E686AADE4F34750101613CD0FA': 'effectEnable3',
                    '8F630FFFC3B987A3695FFADEDE2A0697': 'ipAddress3',
                    '70067A35A2669DBA088AFC924766D490': 'ipAccessMode3',
                    'C486BB39E219128FF89DB3B81B8269B8': 'netmask3',
                    '9438C9C57B8C1D487197576DD1E0433C': 'option603',
                    '7A7D698F3189D5FE2280AD43004FE746': 'ipv6Gateway3',
                    '165A6B2DE8DB6A5943CA41961AF6F0E2': 'ipv6IpAddress3',
                    '09BA33F31249D16F6AA6D0BB7FF225DB': 'prefix3',
                    '434135EACB4605A6DE00CC5A08789CEC': 'vlanId3',

                    '5CE5A72ED22A5DC52BA7E1A387DE7A7B': 'gateway4',
                    '8782A41A33D8251542BEC1A557C0EBA5': 'effectEnable4',
                    '5C686166CBB25EF4E81C1E941BA0CF34': 'ipAddress4',
                    '1510355C44FC831A54CD18EA1A22E506': 'ipAccessMode4',
                    '31B40D58B3FBE2A5608DF5D2AA04AF95': 'netmask4',
                    '9129EEE26D44B9EF7DA9AFF294A4641D': 'option604',
                    '1ED00A6D8C1EFFABD7E60FD382781D29': 'ipv6Gateway4',
                    'BFFC26D46BD53F97C0B4DB1531074ACA': 'ipv6IpAddress4',
                    'D9446CA767C18460BF5489B6359C3DD4': 'prefix4',
                    '2D074E171A883CCB0CC9645036D8E864': 'vlanId4',

                    '11F74E881208316AADEDC4E439732F98': 'gateway5',
                    'DFAA0E293BFDE8C3ECEF5B0DF220BABA': 'effectEnable5',
                    '9EF125792D27FADF13D8E61D18EFC717': 'ipAddress5',
                    '2F858B4CA8E9CFB28F8AB6A1F43972EE': 'ipAccessMode5',
                    '304E06E027B0A382EE9249169D9BDDA7': 'netmask5',
                    '804E329691CE8801673A1939CB2DCA3A': 'option605',
                    '001E2593077F8428B9D47442FFC566E4': 'ipv6Gateway5',
                    'E2B9766065FF6B1174A2405031BF3C4B': 'ipv6IpAddress5',
                    '9B42006741147678102329B5EF1547D1': 'prefix5',
                    'C4F463A1C66B93582C9E142F298E7551': 'vlanId5',

                    'CD2FEF783CF3C8A3F68F51BBFAE1639B': 'gateway6',
                    'A4F1777EA61F9DCCA7004D75A34AB317': 'effectEnable6',
                    '4790FFC2CBFEDF340A1F922203144C3F': 'ipAddress6',
                    'EE8C0AD627522D165456AEE3969FDEC5': 'ipAccessMode6',
                    '015BAE7A8E05F0985EE7E25D3995F228': 'netmask6',
                    'ED62AF8F79A0F7B0DF9AF2E019BF2E5E': 'option606',
                    '93C70A0E804A98BF178D6CB0A3A084F5': 'ipv6Gateway6',
                    '8B57153145C6FCF494286C1E902D1EFD': 'ipv6IpAddress6',
                    '1FE26E5FDFCA1B32C8828FDB9B31961A': 'prefix6',
                    'CAFD8D94BED65B38B4282706C4472575': 'vlanId6',

                    '152E573BB9AFB0A0330ECD681FDEB56A': 'gateway7',
                    'BECF81485EB7EDB1031D5186B8EE5F78': 'effectEnable7',
                    'DFEDC60B0D15908F44FE070E3B444984': 'ipAddress7',
                    'FB7B5324B8DE80BAFC151E3B037A7615': 'ipAccessMode7',
                    '14394F779BB363A725CE424FC813D6AF': 'netmask7',
                    '45667E62D5D8B938D9840200A69A562A': 'option607',
                    '78E0622940D88BC870DE716A171B1A4A': 'ipv6Gateway7',
                    '9EBFA583AA1DE9891578EACF53761864': 'ipv6IpAddress7',
                    '872D3DD424DFCAC4A5C109FAABD6E13B': 'prefix7',
                    '427D8BEF81ACC1A3434B143EA35EF9AD': 'vlanId7',

                    '04B99EEDFDC3D4679784CB7BE2A43009': 'gateway8',
                    'F44A24D697160B488FEBFADBA73E75D9': 'effectEnable8',
                    'F2CF05B79F4959ABD632D83FC7785D1B': 'ipAddress8',
                    'EE4FD9A6AF4A0DDEB5ED5CF003B837A0': 'ipAccessMode8',
                    '6A5B8FAD420F58715E9165B6116BA137': 'netmask8',
                    '668AE26882F8D2D43E7A401D89D3D31D': 'option608',
                    'ABF8631833F548A2A4F62CD305E593A7': 'ipv6Gateway8',
                    '7D31A6EFE34469C83E4ED0C3C57CE1DB': 'ipv6IpAddress8',
                    '1D0590F35E1D831ED883FA933A1EB3B8': 'prefix8',
                    '98D32E22FB4155B79AD524C8A594830E': 'vlanId8',

                    '2D0FE3315183F2E1EBA6C641169A511F': 'gateway9',
                    'E414B3371C8B87B4A826154189781925': 'effectEnable9',
                    '4E084BE3B8C1DA317C4AEC03D61BA522': 'ipAddress9',
                    '2825394E5620DE7D94AA6C17A2C1A6F3': 'ipAccessMode9',
                    'ABB508A1AC38F6049DCF152E5F19E481': 'netmask9',
                    'B8D931A2D5D1BED8BBA652A8E7FFFCC2': 'option609',
                    '84EC047FBE8BD407E9A6526C9D720DCE': 'ipv6Gateway9',
                    '0C0483FE0C1453AA194C23C311DBFC4F': 'ipv6IpAddress9',
                    '239FEC0718B22FBDEEF5EABAD741B5BB': 'prefix9',
                    '088E94172FD2A9E465CC757750858559': 'vlanId9',

                    '6CB9DF8D982F931CD56B6EFA4E69638C':'ipAccessMode10', //第十组
                    '0260F67ADDCD3B1D5D885EBB24518574':'ipAddress10',
                    'AB2D6E2B155148CEB9F2ACE774E511C9':'netmask10',
                    '8BD5455EDBDA7C09048A68B48D5920DD':'gateway10',
                    'B2BCD3C2447AAB1BB916D3A2C3D9FE2A':'prefix10',
                    'B71D4C2F52886AD26856B8FE459CDFE2':'ipv6Gateway10',
                    '687BC8364D775AAF5F937A3F695AC5F6':'ipv6IpAddress10',
                    'C7F464266971F970A7F770EABD0D5E1C':'vlanId10',
                    '0C5742E98211B83DC313A9FECE8E1F3D':'effectEnable10',
                    '12D600E1B50F44544F2391F1CCD42A1B':'option6010',

                    '31D46077A97359AA52752E78B61C6650':'ipAccessMode11', //第11组
					'27CBEF8B1517AD6A5E843B89A89238D8':'ipAddress11',
					'B600801E6F8EBB6A635051A792363416':'netmask11',
					'9AA33718E8D93DC7F0245EC672B02884':'gateway11',
					'3358D81F49C855DB67D65065C20F236D':'prefix11',
                    'A157070914D8736D8126B23480116E24':'ipv6Gateway11',
                    'AC76524D037D92B88665D7F04FA1880F':'ipv6IpAddress11',
					'8D713886502A26B722DB09FBB6D7319D':'vlanId11',
                    'D8744D525D58501AF4C445994E8F9F24':'effectEnable11',
					'C6AD08D8C86F20F2E45AAE854813C608':'option6011',

					'8B04EFD523302351BE4518DEFB8E38FF':'ipAccessMode12', //第12组
                    'B3F45C8A9E15B84CAD319BC38091D522':'ipAddress12',
                    '357A3E78D7B138B15F2206A0FA1B1413':'netmask12',
                    'D486482330F219B5ADD795D0D79DDC73':'gateway12',
                    'E1D7911DDCF1409F6D51EBC2AC6C774F':'prefix12',
                    '596785743C1F9DCA95F621E593E30113':'ipv6Gateway12',
                    '15BAD3BA814F8F854BB7E9218B49FC10':'ipv6IpAddress12',
                    '01C8C43D898E9F094E0FE522483278EF':'vlanId12',
                    'D4606B77BB82E1401F3EFB4CC15FDEC7':'effectEnable12',
                    'AFD2C99B366FCF5D4964BB60C5C11C3D':'option6012',

                    'A3834E731D31DA415FE0B4347B690BF9': 'connectType'
				},
                addIpsecLoading:false
				
			}
		},
		watch: {
            lgwBindGroup:function(){
				this.networkForm.lgwImsiRange = this.lgwBindGroup.toString();
			},
			networkForm: {
				handler: function(newVal, oldVal) {
					var vm = this,
						form = vm.$refs.networkForm;
					
					detectReboot(form, vm.rebootMap);
				},
				deep: true
			},
            
            wanConfigTableList(){
				var vm = this, curValue ='', arr = [];

				if(vm.wanConfigTableList.length != 0){
                    if(!['Intel_CR_CA','Intel_CR_SC','Intel_CR_TC','Intel_CR_DC','Intel_CR','MLN_SC','MLN_CA','MLN_DC','MLN'].includes(vm.platformType)){
                        //映射 quick interface Binding 的值
                        vm.wanConfigTableList.map((item)=>{
                            //fiber: eth0  copper: eth1
                            if(vm.networkForm.connectType == 'fiber'){ 
                                curValue = 'eth0';                             
                            }else{
                                curValue = 'eth1';  
                            } 
                            //wanConfig 第一行数据无开关，但是依旧需要匹配对应的下拉值；
                            if(item.index == '1'){
                                if(item.vlanId === '' || item.vlanId === null || item.vlanId === undefined || item.vlanId === '0'){
                                    arr.push({ label: 'WAN' + item.index, value: curValue + ':' + item.index  })
                                }else{
                                    //connectType值+各行的 VLAN id+对应行号：组合成value 值
                                    arr.push({ label: 'WAN' + item.index + '-VLAN', value: curValue + '.' + item.vlanId + ':' + item.index  })
                                }        
                            }
                            //开关状态是开时，才会显示对应的下拉值；
                            if(item.effectEnable == '1'){
                                // 该条件下 将不显示 -vlan                            
                                if(item.vlanId === '' || item.vlanId === null || item.vlanId === undefined || item.vlanId === '0'){
                                    arr.push({ label: 'WAN' + item.index, value: curValue + ':' + item.index  })
                                }else{
                                    //connectType值+各行的 VLAN id+对应行号：组合成value 值
                                    arr.push({ label: 'WAN' + item.index + '-VLAN', value: curValue + '.' + item.vlanId + ':' + item.index  })
                                }
                            }  
                                                                                             
                            //更新表格展示数据
                            if(item.ipAccessMode == '4'){                         
                                item.ipv6IpAddress = item.ipv6IpAddress;
                                item.prefix = item.prefix;
                                item.ipv6Gateway = item.ipv6Gateway;
                            }else{
                                //ip access mode: '',0,1,3
                                item.ipAddress =  item.ipAddress;
                                item.netmask =  item.netmask;
                                item.gateway = item.gateway
                            }
                        });
                        vm.quickInterfaceBindingList = arr;
                            
                        var obj = { label: 'WAN', value: curValue };
                        vm.quickInterfaceBindingList.unshift(obj); 
                    }else{
                        vm.wanConfigTableList.map((item)=>{
                            //ip access mode: '',0,1,
                            item.ipAddress =  item.ipAddress;
                            item.netmask =  item.netmask;
                            item.gateway = item.gateway;   
                        });                      
                    }                   
				}
			},
		},
		methods:{
			initReboot(list, map) {
				var vm = this;
				
				list.map(function(item){
					if(item.reboot == '1') {
						var code = item.name,
							key = vm.casts[code];

						map[key] = true;
					}
				});
			},
			init(code,id, platformType){
				var vm = this;
				vm.smallCellCode = code;
                if(['Intel_CR_CA','Intel_CR_SC','Intel_CR_TC','Intel_CR_DC','Intel_CR','MLN_SC','MLN_CA','MLN_DC','MLN'].includes(platformType)){
                    vm.wanConfigNumber = 4;
                }else{
                    vm.wanConfigNumber = 12;
                }
                vm.platformType = platformType;
				vm.getParamNode(code,id);
			},
			getParamNode(code,id) {
                var vm = this,
                    codes = [],
                    url = '${ctx}/cell/quicksettings/getParamNodeTreeAndData.action',
                    params = {
                        id: id,
                        smallCellCode: code
                    };

                axios.post(url, stringify(params)).then(function(res){
                    var data = res.data;

                    if(data && Array.isArray(data)) {
                        data.map(function(item){
                            item.groups.map(function(group){
                                group.list.map(function(m){
                                	codes.push(m.name);
                                	if(m.type == 'list'){ 
										m.list.map(function(field){
											vm.codeTableList.push(field.name);
										});
                                		vm.initTable(m.url);
                                	}else{
                                   		// 执行赋值
                                        vm.setValue(m);
                                	}
                                
                                     //lgw imsi to ip binding range
                                    if(m.name == '9F497D0576E74E94EB6C68F4E7C7CADA' && m.value) {
                                        m.value.split(',').map(function(v){
                                            vm.lgwBindGroup.push(v);
                                        });
                                    }

                                    if('A4E3C35055EBB35B2EFA0ECCAC18FFE7' == m.name) {
                                        if(m.data) {
                                            var bandList = JSON.parse(m.data);

                                            vm.newQuickBandList = bandList;
                                        }
                                    }
                                    if('621BD8BE5FA1C1BE22FA36240804B0AA' == m.name) {
                                        if(m.data) {
                                            var bandList = JSON.parse(m.data);

                                            vm.slaveInterfaceList = bandList;
                                        }
                                    }
                                });
								// 初始化重启项关系记录
								vm.initReboot(group.list, vm.rebootMap);
                            });
                        });

                        //组合 wan config 模块的数据
                        if(['Intel_CR_CA','Intel_CR_SC','Intel_CR_TC','Intel_CR_DC','Intel_CR','MLN_SC','MLN_CA','MLN_DC','MLN'].includes(vm.platformType)){
                            ['1','2','3','4'].map(function(item){
                                var row = {},
                                    ipAccessModeKey = 'ipAccessMode' + item,
                                    
                                    ipAddressKey = 'ipAddress' + item,
                                    netmaskKey = 'netmask' + item,
                                    gatewayKey = 'gateway' + item,
    
                                    vlanIdKey = 'vlanId' + item,
                                    effectEnableKey = 'effectEnable' + item,
                                    option60Key = 'option60' + item;
    
                                row['index'] = item;
                                row['wanName'] = 'wanConfig' + item;
                                row['ipAccessMode'] = vm.networkForm[ipAccessModeKey] ||'';
                                row['option60'] = vm.networkForm[option60Key] ||'';
                                row['effectEnable'] = vm.networkForm[effectEnableKey] ||'';    

                                //ip access mode : '',DHCP-0,                             
                                //row['ipAddress'] = vm.networkForm[ipAddressKey] ||'';
                                if(vm.networkForm[ipAddressKey] == '0.0.0.0' || vm.networkForm[ipAddressKey] == '' || vm.networkForm[ipAddressKey] == '::'){
                                    row['ipAddress'] = '';
                                    vm.networkForm[ipAddressKey] = '';
                                }else{
                                    row['ipAddress'] = vm.networkForm[ipAddressKey];
                                }
                                //row['netmask'] = vm.networkForm[netmaskKey] ||'';
                                if(vm.networkForm[netmaskKey] == '0.0.0.0' || vm.networkForm[netmaskKey] == ''){
                                    row['netmask'] = '';
                                    vm.networkForm[netmaskKey] = '';
                                }else{
                                    row['netmask'] = vm.networkForm[netmaskKey];
                                }
                                //row['gateway'] = vm.networkForm[gatewayKey] ||'';
                                if(vm.networkForm[gatewayKey] == '0.0.0.0' || vm.networkForm[gatewayKey] == '' || vm.networkForm[gatewayKey] == '::'){
                                    row['gateway'] = '';
                                    vm.networkForm[gatewayKey] = '';
                                }else{
                                    row['gateway'] = vm.networkForm[gatewayKey];
                                }
                                //row['vlanId'] = vm.networkForm[vlanIdKey] ||'';
                                if(vm.networkForm[vlanIdKey] == '0' || vm.networkForm[vlanIdKey] == ''){
                                    row['vlanId'] = '';
                                    vm.networkForm[vlanIdKey] = '';
                                }else{
                                    row['vlanId'] = vm.networkForm[vlanIdKey];
                                }
                                vm.wanConfigTableList.push(row);
                            })
                        }else{
                            ['1','2','3','4','5','6','7','8','9','10','11','12'].map(function(item){
                                var row = {},
                                    ipAccessModeKey = 'ipAccessMode' + item,
                                    
                                    ipAddressKey = 'ipAddress' + item,
                                    netmaskKey = 'netmask' + item,
                                    gatewayKey = 'gateway' + item,
    
                                    prefixKey = 'prefix' + item,
                                    ipv6IpAddressKey = 'ipv6IpAddress' + item,
                                    ipv6GatewayKey = 'ipv6Gateway' + item,
    
                                    vlanIdKey = 'vlanId' + item,
                                    effectEnableKey = 'effectEnable' + item,
                                    option60Key = 'option60' + item;
    
                                row['index'] = item;
                                row['wanName'] = 'wanConfig' + item;
                                row['ipAccessMode'] = vm.networkForm[ipAccessModeKey] ||'';
                                row['option60'] = vm.networkForm[option60Key] ||'';
                                row['effectEnable'] = vm.networkForm[effectEnableKey] ||'';    
                                
                                //ip access mode - ipv6 static ip -4   
                                if(row['ipAccessMode'] == '4'){
                                    //row['ipv6IpAddress'] = vm.networkForm[ipv6IpAddressKey] ||'';
                                    if(vm.networkForm[ipv6IpAddressKey] == '0.0.0.0' || vm.networkForm[ipv6IpAddressKey] == '' || vm.networkForm[ipv6IpAddressKey] == '::'){
                                        row['ipv6IpAddress'] = '';
                                        vm.networkForm[ipv6IpAddressKey] = '';
                                    }else{                                   
                                        row['ipv6IpAddress'] = vm.networkForm[ipv6IpAddressKey];
                                    }
                                    //row['prefix'] = vm.networkForm[prefixKey] ||'';
                                    if(vm.networkForm[prefixKey] == '0.0.0.0' || vm.networkForm[prefixKey] == ''){
                                        row['prefix'] = '';
                                        vm.networkForm[prefixKey] = '';
                                    }else{
                                        row['prefix'] = vm.networkForm[prefixKey];
                                    }
                                    //row['ipv6Gateway'] = vm.networkForm[ipv6GatewayKey] ||'';
                                    if(vm.networkForm[ipv6GatewayKey] == '0.0.0.0' || vm.networkForm[ipv6GatewayKey] == '' || vm.networkForm[ipv6GatewayKey] == '::'){
                                        row['ipv6Gateway'] = '';
                                        vm.networkForm[ipv6GatewayKey] = '';
                                    }else{
                                       row['ipv6Gateway'] = vm.networkForm[ipv6GatewayKey];
                                    }
                                    //普通三组
                                    row['ipAddress'] = vm.networkForm[ipAddressKey];
                                    row['netmask'] = vm.networkForm[netmaskKey];
                                    row['gateway'] = vm.networkForm[gatewayKey];
                                }else{
                                    //ip access mode : '',DHCP-0,Static IP-1,IPV6 DHCP-3                              
                                    //row['ipAddress'] = vm.networkForm[ipAddressKey] ||'';
                                    if(vm.networkForm[ipAddressKey] == '0.0.0.0' || vm.networkForm[ipAddressKey] == '' || vm.networkForm[ipAddressKey] == '::'){
                                        row['ipAddress'] = '';
                                        vm.networkForm[ipAddressKey] = '';
                                    }else{
                                       row['ipAddress'] = vm.networkForm[ipAddressKey];
                                    }
                                    //row['netmask'] = vm.networkForm[netmaskKey] ||'';
                                    if(vm.networkForm[netmaskKey] == '0.0.0.0' || vm.networkForm[netmaskKey] == ''){
                                        row['netmask'] = '';
                                        vm.networkForm[netmaskKey] = '';
                                    }else{
                                        row['netmask'] = vm.networkForm[netmaskKey];
                                    }
                                    //row['gateway'] = vm.networkForm[gatewayKey] ||'';
                                    if(vm.networkForm[gatewayKey] == '0.0.0.0' || vm.networkForm[gatewayKey] == '' || vm.networkForm[gatewayKey] == '::'){
                                        row['gateway'] = '';
                                        vm.networkForm[gatewayKey] = '';
                                    }else{
                                        row['gateway'] = vm.networkForm[gatewayKey];
                                    }
    
                                    // ipv6 三组
                                    row['ipv6Gateway'] = vm.networkForm[ipv6GatewayKey];
                                    row['prefix'] = vm.networkForm[prefixKey];
                                    row['ipv6IpAddress'] = vm.networkForm[ipv6IpAddressKey];
                                }
                               
                                //row['vlanId'] = vm.networkForm[vlanIdKey] ||'';
                                if(vm.networkForm[vlanIdKey] == '0' || vm.networkForm[vlanIdKey] == ''){
                                    row['vlanId'] = '';
                                    vm.networkForm[vlanIdKey] = '';
                                }else{
                                    row['vlanId'] = vm.networkForm[vlanIdKey];
                                }
                                vm.wanConfigTableList.push(row);
                            })
                        }

                        // 组合 Static Routing 模块数据
                        if(['Intel_CR_CA','Intel_CR_SC','Intel_CR_TC','Intel_CR_DC','Intel_CR','MLN_SC','MLN_CA','MLN_DC','MLN'].includes(vm.platformType)){ // 4860
                            ['1','2','3','4'].map(function(item) {
                                var row = {},
                                    indexKey = 'routeIndex' + item,
                                    effectKey = 'routeEffective' + item,
                                    networkKey = 'routeNetwork' + item,
                                    netmaskKey = 'routeNetmask' + item,
                                    gatewayKey = 'routeGateway' + item;

                                //row['routeIndex'] = vm.networkForm[indexKey] || '';
                                row['routeEffective'] = vm.networkForm[effectKey] || '';
                                row['routeNetwork'] = vm.networkForm[networkKey] || '';
                                row['routeNetmask'] = vm.networkForm[netmaskKey] || '';
                                row['routeGateway'] = vm.networkForm[gatewayKey] || '';

                                vm.routerList.push(row);
                            });
                        }else {
                            ['1','2','3','4','5','6','7','8','9','10','11','12'].map(function(item) {
                                var row = {},
                                    indexKey = 'routeIndex' + item,
                                    effectKey = 'routeEffective' + item,
                                    networkKey = 'routeNetwork' + item,
                                    netmaskKey = 'routeNetmask' + item,
                                    gatewayKey = 'routeGateway' + item;

                                //row['routeIndex'] = vm.networkForm[indexKey] || '';
                                row['routeEffective'] = vm.networkForm[effectKey] || '';
                                row['routeNetwork'] = vm.networkForm[networkKey] || '';
                                row['routeNetmask'] = vm.networkForm[netmaskKey] || '';
                                row['routeGateway'] = vm.networkForm[gatewayKey] || '';

                                vm.routerList.push(row);
                            });
                        }
                        
                        vm.$nextTick(function(){
                            initForm(vm.$refs.networkForm);
                            detectReboot(vm.$refs.networkForm, vm.rebootMap);
							$('#setting_main').removeClass('loading');
                        });
                        vm.codeList = codes;
                    }
                });
            },
            setValue(item,type) {
            	var vm = this,
                code = item.name,
                value = item.value;

	            // indexs是否含有
	            var key = vm.casts[code];
	            if (key){
	            	vm.networkForm[key] = value;
	            }
	            
            },
            initTable(url){
            	var vm = this,
					params = {
            			smallCellCode : vm.smallCellCode
            		};
				
            	axios.post(url,stringify(params)).then(res=>{
            		var data = res.data;
            		if(data.rows){
            			data.rows.map(item=>{
            				var obj = {};
            				for(var key in item){
            					obj[vm.casts[key]] = item[key]
            				}
            				vm.networkForm.ipsecList.push(obj);
							vm.oldTbList.push(Object.assign({},obj));
            			})
            		}
            	})
            },
            getNameByProp(prop) {
                var vm = this,
                    reg = /^\w*\.\d*\.\w*$/,
                    key = prop;
                
                if(reg.test(prop)) {
                    var mReg = /\.(\d*)\./,
                        sufReg = /\.(\w*)$/,
                        idx = prop.match(mReg)[1],
                        sufStr = prop.match(sufReg)[1];

                    vm.codeList.map(function(name){
                        var index = vm.indexs[name];
                        if(vm.casts[name] == sufStr && index == idx) {
                            key = name;
                        }
                    });
                }else {
                    (vm.codeList.concat(vm.codeTableList)).map(function(name){
                        if(vm.casts[name] == prop) {
                            key = name;
                        }
                    });
                }

                return key;
            },
            isNull(val){
                if(val==undefined || val == null || val =="") return true;
                else return false;
            },
			addIpsec(){
				var vm = this;
                $('#ipsecAddPanel').html('');
				vm.showIpsecAdd = true;
				vm.operType = 'add';
                vm.addIpsecLoading = true;
				loadHTML(document.querySelector('#ipsecAddPanel'),{
                    url:'${ctx}/cell/quicksettings/goIPSecParamPage.action',
                    success: function() {
                        vm.addIpsecLoading = false
                    }
                });
			},
			closeIpsec(){
				this.showIpsecAdd = false;
			},
			save(){
				var vm = this;
                if(vm.addIpsecLoading)return
				if(vm.showIpsecAdd){
					eventBus.$emit('save-ipsec');
				}else{
	                var params = {},
	                    isChanged = isFormChanged(vm.$refs.networkForm);

	                if(!isChanged){
	                    showMsg('prompt_msg','<%=rb.getString("CanShuZhiMeiYouBianHua")%>');
	                    return;
	                }

	                vm.$refs.networkForm.fields.map(function(field){
	                    var key = vm.getNameByProp(field.prop);

	                    if(Array.isArray(field.fieldValue)){
	                        var vList = field.fieldValue.map(function(item){return item}),
	                            oList = (field.reinitialValue||[]).map(function(item){return item}),
	                            val = JSON.stringify(vList.sort()),
	                            orVal = JSON.stringify(oList.sort());

	                        if(val != orVal) {
	                            params[key] = val;

								if(['ipsecList'].includes(field.prop)) {
									var nList = [];
									vList.map(function(m){
										var obj = {},
											rowMatcheds = vm.oldTbList.filter(function(row){
												return row.index == m.index;
											}),
											mRow = rowMatcheds.length?rowMatcheds[0]:'';

										for(k in m) {
											var prop = vm.getNameByProp(k);

											if(mRow) {
												if(mRow[k] != m[k] || k=='index') obj[prop] = m[k];
											}else {
												obj[prop] = m[k];
											}
										}
										nList.push(obj);
									});

									params[key] = nList.filter(function(item){
										return ['edit','add'].includes(item.operateType);
									});

                                    if(vm.delRecord[field.prop].length) {
										vm.delRecord[field.prop].map(function(im){
											var obj = {};

											for(k in im) {
												var prop = vm.getNameByProp(k);

												obj[prop] = im[k];
											}
											
											params[key].push(obj);
										})
									}
								}
	                        };
	                    }else{
	                        if(vm.isNull(field.fieldValue) && vm.isNull(field.reinitialValue)){
	                            
	                        }else if(field.fieldValue != field.reinitialValue) {
	                            params[key] = field.fieldValue;
	                        };
	                    }
	                });
	                
	                vm.$refs.networkForm.validate(function(valid){
	                    if(valid) {
							var isNeedReboot = detectReboot(vm.$refs.networkForm, vm.rebootMap);

							if(isNeedReboot) {
								var tipContent = [
										'<%=rb.getString("JiZhanChongQiTiShi")%>',
										'<br/><br/>',
										'<input id="reboot_confirm_status" type="checkbox" />',
										'<label for="reboot_confirm_status" style="font-size: 14px;color: #1DA3FC;cursor: pointer;"><%=rb.getString("SheZhiHouChongQi")%></label>'
									].join(" ");

								var msger = $.messager.confirm('<%=rb.getString("QueRen")%>', tipContent, function (r) {
									if(r) {
										/* 重启勾选判断 */
										var needReboot = false,
											rebootCkbox = $('#reboot_confirm_status',msger);
										if(rebootCkbox.length && rebootCkbox.prop('checked')){
											needReboot = true;
										}
										msger = null;
										
										var rowCode = vm.smallCellCode,
											url = '${ctx}/cell/quicksettings/saveParamValue.action?smallCellCode='+rowCode;

										$('#setting_main').addClass('loading');
										settingVue.submitDisabled = true;
										axios.post(url,stringify({"params": JSON.stringify(params)})).then(res=>{
											var data = res.data;
											if(data["success"]){
												vm.$message.success({type:'success',message:'<%=rb.getString("ChengGong")%>'});

												// 勾选重启，下发重启指令
												if(needReboot) {
													$.post("${ctx}/cell/cpeinfos/cellReboot.action", {cell_code: rowCode}, function (data) {
														if (!data["success"]) {
															showMsg('error_msg',data["message"]);
														}
													}, "json");
												}
												closeSettingPanel();
											}else{
												vm.$message.error(data["message"])
											}

											$('#setting_main').removeClass('loading');
										    settingVue.submitDisabled = false;
										})
									}
								}).addClass("seriousConfirm");
							} else {
								var rowCode = vm.smallCellCode,
									url = '${ctx}/cell/quicksettings/saveParamValue.action?smallCellCode='+rowCode;
										
                                $('#setting_main').addClass('loading');
								settingVue.submitDisabled = true;
								axios.post(url,stringify({"params": JSON.stringify(params)})).then(res=>{
									var data = res.data;
									if(data["success"]){
										vm.$message.success({type:'success',message:'<%=rb.getString("ChengGong")%>'});
										closeSettingPanel();
									}else{
										vm.$message.error(data["message"])
									}

                                    $('#setting_main').removeClass('loading');
									settingVue.submitDisabled = false;
								})
							}
						}
	                });
				}
			},
			ipsecFmt(row,column,value){
                if([null,undefined].includes(value)) {
                    return '';
                }else {
				    return value.toString();
                }
			},
			editIpsec(row){
                var vm = this;
				$('#ipsecAddPanel').html('');
				vm.showIpsecAdd = true;
				vm.operType = 'edit';
				vm.rowData = row;
				vm.addIpsecLoading = true;
				loadHTML(document.querySelector('#ipsecAddPanel'),{
                    url:'${ctx}/cell/quicksettings/goIPSecParamPage.action',
                    success: function() {
                        vm.addIpsecLoading = false
                    }
                });
			},
			delIpsec(row){
				var vm = this;
				var confirmStr = '<%=rb.getString("QueRenShanChu")%>'
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(()=>{
					var ipsecArr = vm.networkForm.ipsecList.map(function(item){
						return item.index;
					})
					var index = ipsecArr.indexOf(row.index);
					vm.networkForm.ipsecList.splice(index,1);
					var length = vm.networkForm.ipsecList.length;
					for(let i=0;i<length;i++){
						vm.networkForm.ipsecList[i].index = i+1;
					}
					
					row.operateType = 'remove';
					vm.delRecord['ipsecList'].push(Object.assign({operateType: 'remove'},{index: row.index+''}));
				})
			},
			cancel(){
				var vm = this;
				if(isFormChanged(vm.$refs.networkForm)){//返回true为改变
					vm.$confirm('<%=rb.getString("QueDingLiKaiDangQianYeMian")%>','<%=rb.getString("QueRen")%>',{
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						cancelButtonText:'<%=rb.getString("QuXiao")%>',
						type:'warning',
						closeOnClickModal:false
					}).then(() => {
						closeSettingPanel();
					}).catch(() => {})
				}else{
					closeSettingPanel();
				}
			},

            //wan config  点击修改操作
            addWANDialogOpen(row){
                var vm = this;

                Object.assign(vm.addWanConfigForm,row)
                vm.addWanConfigDialogShow = true;
            },
            addWanConfigDialogSubmit(){
                var vm = this, curValue= '', arr = [];

                vm.$refs.addWanConfigForm.validate(function(valid){
                    if(valid){
                        if(!['Intel_CR_CA','Intel_CR_SC','Intel_CR_TC','Intel_CR_DC','Intel_CR','MLN_SC','MLN_CA','MLN_DC','MLN'].includes(vm.platformType)){
                            vm.wanConfigTableList.map((item)=>{
                                if(item.index == vm.addWanConfigForm.index) {
                                    //同步form 数据
                                    Object.assign(item, vm.addWanConfigForm);
                                    //更新表格展示数据
                                   if(item.ipAccessMode == '4'){
                                        item.ipv6IpAddress = item.ipv6IpAddress;
                                        item.prefix = item.prefix;
                                        item.ipv6Gateway = item.ipv6Gateway;
                                    }else{
                                        //ip access mode: '',0,1,3
                                        item.ipAddress =  item.ipAddress;
                                        item.netmask =  item.netmask;
                                        item.gateway = item.gateway;
                                    }
                                } 
                                //fiber: eth0  copper: eth1
                                if(vm.networkForm.connectType == 'fiber'){ 
                                    curValue = 'eth0';                             
                                }else{
                                    curValue = 'eth1';  
                                } 
                                //wanConfig 第一行数据无开关，但是依旧需要匹配对应的下拉值；
                                if(item.index == '1'){
                                    if(item.vlanId === '' || item.vlanId === null || item.vlanId === undefined || item.vlanId === '0'){
                                        arr.push({ label: 'WAN' + item.index, value: curValue + ':' + item.index  })
                                    }else{
                                        //connectType值+各行的 VLAN id+对应行号：组合成value 值
                                        arr.push({ label: 'WAN' + item.index + '-VLAN', value: curValue + '.' + item.vlanId + ':' + item.index })
                                    }        
                                }
                                //开关状态是开时，才会显示对应的下拉值；
                                if(item.effectEnable == '1'){
                                    // 该条件下 将不显示 -vlan                            
                                    if(item.vlanId === '' || item.vlanId === null || item.vlanId === undefined || item.vlanId === '0'){
                                        arr.push({ label: 'WAN' + item.index, value: curValue + ':' + item.index  })
                                    }else{
                                        //connectType值+各行的 VLAN id+对应行号：组合成value 值
                                        arr.push({ label: 'WAN' + item.index + '-VLAN', value: curValue + '.' + item.vlanId + ':' + item.index  })
                                    }
                                }
                                //当前的行数据与form绑定的prop 做对比 重新赋值
                                if(['1','2','3','4','5','6','7','8','9','10','11','12'].includes(item.index)) {
                                    var index = item.index,
                                        ipAccessModeKey = 'ipAccessMode' + index,
                                        ipAddressKey = 'ipAddress' + index,
                                        netmaskKey = 'netmask' + index,
                                        gatewayKey = 'gateway' + index,
                                        prefixKey = 'prefix' + index,
                                        ipv6IpAddressKey = 'ipv6IpAddress' + index,
                                        ipv6GatewayKey = 'ipv6Gateway' + index,
                                        vlanIdKey = 'vlanId' + index,
                                        effectEnableKey = 'effectEnable' + index,
                                        option60Key = 'option60' + index;
                                
                                    // networkForm表单对应数据更新
                                    vm.networkForm[ipAccessModeKey] = item['ipAccessMode'];
                                    vm.networkForm[ipAddressKey] = item['ipAddress'];
                                    vm.networkForm[netmaskKey] = item['netmask'];
                                    vm.networkForm[gatewayKey] = item['gateway'];
                                    vm.networkForm[prefixKey] = item['prefix'];
                                    vm.networkForm[ipv6IpAddressKey] = item['ipv6IpAddress'];
                                    vm.networkForm[ipv6GatewayKey] = item['ipv6Gateway'];
                                    vm.networkForm[vlanIdKey] = item['vlanId'];
                                    vm.networkForm[effectEnableKey] = item['effectEnable'];
                                    vm.networkForm[option60Key] = item['option60'];
                                }
                            });
                            vm.quickInterfaceBindingList = arr;
                                
                            var obj = { label: 'WAN', value: curValue };
                            vm.quickInterfaceBindingList.unshift(obj); 
                        }else{
                            vm.wanConfigTableList.map((item)=>{
                                if(item.index == vm.addWanConfigForm.index) {
                                    Object.assign(item, vm.addWanConfigForm);
                                    item.ipAddress =  item.ipAddress;
                                    item.netmask =  item.netmask;
                                    item.gateway = item.gateway;
                                }
                                //当前的行数据与form绑定的prop 做对比 重新赋值
                                if(['1','2','3','4'].includes(item.index)) {
                                    var index = item.index,
                                        ipAccessModeKey = 'ipAccessMode' + index,
                                        ipAddressKey = 'ipAddress' + index,
                                        netmaskKey = 'netmask' + index,
                                        gatewayKey = 'gateway' + index,
                                    
                                        vlanIdKey = 'vlanId' + index,
                                        effectEnableKey = 'effectEnable' + index,
                                        option60Key = 'option60' + index;
                                
                                    // networkForm表单对应数据更新
                                    vm.networkForm[ipAccessModeKey] = item['ipAccessMode'];
                                    vm.networkForm[ipAddressKey] = item['ipAddress'];
                                    vm.networkForm[netmaskKey] = item['netmask'];
                                    vm.networkForm[gatewayKey] = item['gateway'];                                   
                                    vm.networkForm[vlanIdKey] = item['vlanId'];
                                    vm.networkForm[effectEnableKey] = item['effectEnable'];
                                    vm.networkForm[option60Key] = item['option60'];
                                }
                            });
                        }

                        vm.addWanConfigDialogShow = false;
                    }
                })
            },
            closeAddWanConfigDialog(){
                var vm = this;

                vm.$refs.addWanConfigForm.clearValidate();
                vm.addWanConfigDialogShow = false;
            },
            // Static Routing 修改操作
            routeModifyOpen(row, index){
                var vm = this;

                Object.assign(vm.routeForm, row);
                vm.routeForm.index = index;
                vm.routeDlShow = true;
            },
            routeDlSubmit() {
                var vm = this;

                vm.$refs.routeModify.validate(function(valid) {
                    if(valid) {
                        var index = vm.routeForm.index + 1,
                            indexKey = 'routeIndex' + index,
                            effectKey = 'routeEffective' + index,
                            networkKey = 'routeNetwork' + index,
                            netmaskKey = 'routeNetmask' + index,
                            gatewayKey = 'routeGateway' + index;

                        //vm.networkForm[indexKey] = vm.routeForm['routeIndex'];
                        vm.networkForm[effectKey] = vm.routeForm['routeEffective'];
                        vm.networkForm[networkKey] = vm.routeForm['routeNetwork'];
                        vm.networkForm[netmaskKey] = vm.routeForm['routeNetmask'];
                        vm.networkForm[gatewayKey] = vm.routeForm['routeGateway'];

                        var tbRow = vm.routerList[vm.routeForm.index];
                        tbRow['routeEffective'] = vm.routeForm['routeEffective'];
                        tbRow['routeNetwork'] = vm.routeForm['routeNetwork'];
                        tbRow['routeNetmask'] = vm.routeForm['routeNetmask'];
                        tbRow['routeGateway'] = vm.routeForm['routeGateway'];

                        vm.routeDlShow = false;
                    }
                })
            },
            closeRouteDl() {
                var vm = this;

                vm.$refs.routeModify.clearValidate();
                vm.routeDlShow = false;
            },
            //校验IP
            isValidIP(ip){
                var reg =  /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/
                return reg.test(ip);
            },
            //Ipv6校验
            isIPv6(str){
                var reg = /^([\da-fA-F]{1,4}:){6}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^::([\da-fA-F]{1,4}:){0,4}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:):([\da-fA-F]{1,4}:){0,3}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){2}:([\da-fA-F]{1,4}:){0,2}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){3}:([\da-fA-F]{1,4}:){0,1}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){4}:((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){7}[\da-fA-F]{1,4}$|^:((:[\da-fA-F]{1,4}){1,6}|:)$|^[\da-fA-F]{1,4}:((:[\da-fA-F]{1,4}){1,5}|:)$|^([\da-fA-F]{1,4}:){2}((:[\da-fA-F]{1,4}){1,4}|:)$|^([\da-fA-F]{1,4}:){3}((:[\da-fA-F]{1,4}){1,3}|:)$|^([\da-fA-F]{1,4}:){4}((:[\da-fA-F]{1,4}){1,2}|:)$|^([\da-fA-F]{1,4}:){5}:([\da-fA-F]{1,4})?$|^([\da-fA-F]{1,4}:){6}:$/
                return reg.test(str);
            },
            //校验子网掩码
            isMask(str){
                var exp=/^(254|252|248|240|224|192|128|0)\.0\.0\.0|255\.(254|252|248|240|224|192|128|0)\.0\.0|255\.255\.(254|252|248|240|224|192|128|0)\.0|255\.255\.255\.(254|252|248|240|224|192|128|0)$/;
                return exp.test(str);
            },
            // 验证输入的是否是数字
            isNumeric(str) {
                if(str.length==0){
                    return false;
                }
                for(var i=0;i<str.length;i++){
                    if(str.charAt(i)<"0" || str.charAt(i)>"9"){
                        return false;
                    }
                }
                return true;
            },
            //lgw
            lgwAddBind(){ 
            	var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/,
					vm = this,
					imsi = vm.lgwBindImsi,
					ip = vm.lgwBindIp,
					str = '';

				if(imsi == '' || imsi == ''){
					this.lgwBindCls = 'is-error';
				}else{
					if(reg.test(imsi) && isValidIP(ip) && imsi.length==15 && vm.lgwIsRangeIn()){
						//str = ipStart + "-" + ipEnd;
						vm.lgwBindGroup.push(imsi + '+' + ip);
						vm.lgwBindImsi = '';
						vm.lgwBindIp = '';
						vm.lgwBindCls = '';
					}else{
						vm.lgwBindCls = 'is-error';
					}
				}
            },
            lgwRemoveBind(index){
            	this.lgwBindGroup.splice(index,1);
            },
            lgwIsAllBindIpInRange() {
				var vm = this,
					startIp = vm.networkForm.lgwFirstIp,
					endIp = vm.networkForm.lgwLastIp,
					group = vm.lgwBindGroup,
					bool = true;

				group.map(function(item){
					var ip = item.split('+')[1];

					if(!vm.lgwCompareIp(ip,startIp,endIp)) {
						bool = false;
					}
				});

				if(bool) {
					vm.lgwBindCls = '';
				}else {
					vm.lgwBindCls = 'is-error';
				}

				return bool;
			},
			lgwIsRangeIn() {
				var vm = this,
					ip = vm.lgwBindIp,
					startIp = vm.networkForm.lgwFirstIp,
					endIp = vm.networkForm.lgwLastIp,
					bool = false;

				if(isValidIP(ip) && isValidIP(startIp) && isValidIP(endIp) && vm.lgwCompareIp(ip,startIp,endIp)) {
					bool = true;
				}

				return bool;
			},
            lgwValidateStaticIP(val) {
				var vm = this,
					enable = vm.networkForm.lgwIpEnable == '1';

				if(enable) {
					vm.$refs.networkForm.validateField('lgwFirstIp');
					vm.$refs.networkForm.validateField('lgwLastIp');
				}
			},
            lgwCompareIp(ipvalue,startip,endip) {
			    var vm = this,
			    	ipNum = vm.lgwChangeIpToNum(ipvalue),
			        startNum = vm.lgwChangeIpToNum(startip),
			        endNum = vm.lgwChangeIpToNum(endip);

			    if(isLessThan(ipNum, endNum) && isLessThan(startNum, ipNum)) {
			        return true;
			    }else {
			        return false;
			    }

			    return true;
			},
            lgwChangeIpToNum(ipStr) {
			    var list = (ipStr || '').split('.');

			    list = list.map(function(item){
			        if(item.length < 3) {
			            var dis = 3 - item.length;
			            for(var i = 0; i < dis; i++) item = '0' + item;
			        }

			        return item;
			    });

			    return list.join('');
			},
		},
		mounted(){
			eventBus.$off('close-ipsec').$on('close-ipsec',this.closeIpsec);
			eventBus.$off('save-set').$on('save-set',this.save);
			eventBus.$off('tab-param').$on('tab-param',this.init);
			eventBus.$off('cancel-set-tab').$on('cancel-set-tab',this.cancel);
		}
	})
    //计算静态IP范围的起始值 
	function lgwGetLowAddr(ip, netMask){
	    var lowAddr = "";
	    var ipArray = new Array();
	    var netMaskArray = new Array();
	    
	    if (4 != ip.split(".").length || netMask == ""){
	        return "";
	    }
	    for (var i = 0; i < 4; i++){
	        ipArray[i] = ip.split(".")[i];
	        netMaskArray[i] = netMask.split(".")[i];
	        if ((ipArray[i] > 255) || (ipArray[i] < 0) || (netMaskArray[i] > 255) && (netMaskArray[i] < 0)){
	            return "";
	        }
	        ipArray[i] = ipArray[i] & netMaskArray[i];
	    }
	    
	    for (var i = 0; i < 4; i++){
	        if(i == 3){
	            ipArray[i] = ipArray[i] + 1;
	        }
	        if (lowAddr == ""){
	            lowAddr +=ipArray[i];
	        } else{
	            lowAddr += "." + ipArray[i];
	        }
	    }
	    return lowAddr;
	}

	//计算静态IP范围的终止值 
	function lgwGetHighAddr(ip,netMask){
	    var lowAddr = lgwGetLowAddr(ip,netMask);
	    var hostNumber = lgwGetHostNumber(netMask);
	    if(lowAddr == "" || hostNumber == 0){
	        return "";
	    }
	    
	    var lowAddrArray = new Array();
	    for(var i = 0; i < 4; i++){
	        lowAddrArray[i] = lowAddr.split(".")[i];
	        if(i == 3){
	            lowAddrArray[i] = Number(lowAddrArray[i] - 1);
	        }
	    }
	    lowAddrArray[3] = lowAddrArray[3] + Number(hostNumber - 1);
	   
	    if(lowAddrArray[3] > 255){
	        var k = parseInt(lowAddrArray[3] / 256);       
	        
	        lowAddrArray[3] = lowAddrArray[3] % 256;
	       
	        lowAddrArray[2] = Number(lowAddrArray[2]) + Number(k);       
	       
	        if(lowAddrArray[2] > 255){
	            k = parseInt(lowAddrArray[2] / 256);
	            lowAddrArray[2] = lowAddrArray[2] % 256;
	            lowAddrArray[1] = Number(lowAddrArray[1]) + Number(k);
	            if(lowAddrArray[1] > 255){
	                k = parseInt(lowAddrArray[1] / 256);
	                lowAddrArray[1] = lowAddrArray[1] % 256;
	                lowAddrArray[0] = Number(lowAddrArray[0]) + Number(k);
	            }
	        }
	    }

	    var highAddr = "";
	    for(var i = 0; i < 4; i++){
	        if(i == 3){
	          lowAddrArray[i] = lowAddrArray[i] - 1;
	        }if(highAddr == ""){
	            highAddr = lowAddrArray[i];
	        }else{
	            highAddr += "." + lowAddrArray[i];
	        }
	    }
	    
	    return highAddr;
	}

    function lgwGetHostNumber(netMask){
	    var hostNumber = 0;
	    var netMaskArray = new Array();
	    for(var i = 0; i < 4; i++)
	  {
	        netMaskArray[i] = netMask.split(".")[i];
	        if(netMaskArray[i] < 255)
	    {
	            hostNumber = Math.pow(256,3-i) * (256 - netMaskArray[i]);
	            break;
	        }
	    }

	    return hostNumber;
	}
	

</script>